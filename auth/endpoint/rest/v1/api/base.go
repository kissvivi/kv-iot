package api

type BaseApi struct {
	UserApi UserApiImpl
}

func NewBaseApi(userApi *UserApiImpl) *BaseApi {
	return &BaseApi{UserApi: *userApi}
}
