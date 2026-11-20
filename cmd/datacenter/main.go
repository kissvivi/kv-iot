package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"kv-iot/config"
	"kv-iot/datacenter/configstore"
	dv1 "kv-iot/datacenter/endpoint/http/v1"
	"kv-iot/datacenter/protocol/defaults"
	iothub "kv-iot/datacenter/runtime"
	"kv-iot/datacenter/state"
	"kv-iot/datacenter/storage"
	"kv-iot/db"
)

var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "运行数据中心服务",
	Long:  "运行数据中心服务，加载通信端点、协议驱动、状态存储和运行时 API",
	Run:   runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

var rootCmd = &cobra.Command{
	Use:   "kv-iot",
	Short: "kv-iot 数据中心服务",
}

func runServer(_ *cobra.Command, _ []string) {
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	initDB(cfg)

	registry, err := defaults.NewRegistry()
	if err != nil {
		log.Fatalf("注册协议驱动失败: %v", err)
	}
	log.Printf("已注册协议驱动: %v", registry.ProtocolTypes())

	resolver, err := configstore.NewChannelSpecResolver(db.MYSQLDB)
	if err != nil {
		log.Fatalf("初始化通道配置解析器失败: %v", err)
	}

	var redisExecutor *state.GoRedisExecutor
	var stateStore state.StateStore = state.NewMemoryStore()
	if cfg.Datasource.Redis.Url != "" {
		redisExecutor, err = state.NewGoRedisExecutor(
			cfg.Datasource.Redis.Url,
			cfg.Datasource.Redis.Username,
			cfg.Datasource.Redis.Password,
			cfg.Datasource.Redis.Db,
		)
		if err != nil {
			log.Printf("Redis 配置无效，使用内存状态: %v", err)
		} else {
			pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			pingErr := redisExecutor.Ping(pingCtx)
			cancel()
			if pingErr != nil {
				log.Printf("Redis 连接失败，使用内存状态: %v", pingErr)
				_ = redisExecutor.Close()
				redisExecutor = nil
			} else {
				redisStore, storeErr := state.NewRedisStore(redisExecutor, state.RedisStoreOptions{Prefix: "iot"})
				if storeErr != nil {
					log.Printf("Redis 状态存储初始化失败，使用内存状态: %v", storeErr)
					_ = redisExecutor.Close()
					redisExecutor = nil
				} else {
					stateStore = redisStore
					log.Printf("Redis 状态存储已启用: %s", cfg.Datasource.Redis.Url)
				}
			}
		}
	}
	stateSink, err := iothub.NewStateSampleSink(stateStore, iothub.StateSampleSinkOptions{
		DefaultTTL: secondsTTL(cfg.Runtime.DefaultPollIntervalMs, cfg.Runtime.StateTTLMultiplier),
	})
	if err != nil {
		log.Fatalf("初始化状态 Sink 失败: %v", err)
	}
	historyStore := storage.NewMemoryHistoryStore(20000)
	sinks := []iothub.SampleSink{stateSink, historyStore}
	var telemetryReader storage.SeriesReader = historyStore

	var influxWriter *storage.InfluxWriter
	if cfg.Datasource.Influx.Url != "" && cfg.Datasource.Influx.Org != "" && cfg.Datasource.Influx.Bucket != "" {
		influxWriter, err = storage.NewInfluxWriterFromConnection(
			cfg.Datasource.Influx.Url,
			cfg.Datasource.Influx.Token,
			storage.InfluxWriterConfig{
				Org:    cfg.Datasource.Influx.Org,
				Bucket: cfg.Datasource.Influx.Bucket,
				OnError: func(err error) {
					log.Printf("InfluxDB 写入错误: %v", err)
				},
			},
		)
		if err != nil {
			log.Printf("InfluxDB 不可用，仅使用 Redis/内存状态: %v", err)
		} else {
			pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			pingErr := influxWriter.Ping(pingCtx)
			cancel()
			if pingErr != nil {
				log.Printf("InfluxDB 连接失败，仅使用内存历史: %v", pingErr)
				_ = influxWriter.Close()
				influxWriter = nil
			} else {
				sinks = append(sinks, influxWriter)
				telemetryReader = storage.FallbackReader{
					Primary:   influxWriter,
					Secondary: historyStore,
				}
			}
		}
	}

	manager, err := iothub.NewManager(registry, iothub.NewSinkGroup(sinks...), iothub.WorkerOptions{
		PollInterval:      milliseconds(cfg.Runtime.DefaultPollIntervalMs),
		ReconnectInterval: milliseconds(cfg.Runtime.ChannelReconnectIntervalMs),
	})
	if err != nil {
		log.Fatalf("初始化通道管理器失败: %v", err)
	}

	commandStore := iothub.NewMemoryCommandStore()
	auditStore := iothub.NewMemoryAuditStore()
	commandService, err := iothub.NewCommandService(manager, commandStore, iothub.CommandServiceOptions{
		DefaultTimeout: milliseconds(cfg.Runtime.CommandDefaultTimeoutMs),
		Audit:          auditStore,
	})
	if err != nil {
		log.Fatalf("初始化命令服务失败: %v", err)
	}
	eventStore := iothub.NewMemoryEventStore()

	api, err := dv1.NewAPI(manager, stateStore, commandService, eventStore)
	if err != nil {
		log.Fatalf("初始化运行时 API 失败: %v", err)
	}
	api.SetRuntimeSpecResolver(resolver)
	api.SetPublicDeviceCatalog(resolver)
	api.SetTelemetryReader(telemetryReader)

	if err := startEnabledChannels(context.Background(), resolver, manager); err != nil {
		log.Printf("启动已启用通道时出现错误: %v", err)
	}

	httpServer := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", cfg.Application.DatacenterServer.HttpServer.Host, cfg.Application.DatacenterServer.HttpServer.Port),
		Handler:        dv1.NewRouter(api),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		log.Printf("数据中心 HTTP 服务监听: %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("数据中心 HTTP 服务异常: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭数据中心服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	_ = manager.StopAll(ctx)
	_ = commandService.Close(ctx)
	if influxWriter != nil {
		_ = influxWriter.Close()
	}
	if redisExecutor != nil {
		_ = redisExecutor.Close()
	}
	log.Println("数据中心服务已关闭")
}

func startEnabledChannels(ctx context.Context, resolver *configstore.ChannelSpecResolver, manager *iothub.Manager) error {
	ids, err := resolver.ListEnabledChannelIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		spec, err := resolver.LoadChannelSpec(ctx, id)
		if err != nil {
			log.Printf("加载通道 %s 失败: %v", id, err)
			continue
		}
		if err := manager.StartChannel(ctx, spec); err != nil {
			log.Printf("启动通道 %s 失败: %v", id, err)
			continue
		}
		log.Printf("通道 %s 已启动，点位数: %d", id, len(spec.PollPoints))
	}
	return nil
}

func initDB(cfg *config.Config) {
	baseDB := db.NewBaseDB("mysql")
	baseDB.InitDB(cfg)
	log.Println("数据库连接初始化成功")
}

func secondsTTL(intervalMs, multiplier int) time.Duration {
	if intervalMs <= 0 {
		intervalMs = 3000
	}
	if multiplier <= 0 {
		multiplier = 3
	}
	return milliseconds(intervalMs) * time.Duration(multiplier)
}

func milliseconds(value int) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
