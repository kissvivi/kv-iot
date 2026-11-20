package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrLogin            = errors.New("用户名或密码错误")
	ErrUserNotFound     = errors.New("用户不存在")
	ErrUserExists       = errors.New("用户名已存在")
	ErrUserNameRequired = errors.New("用户名不能为空")
	ErrRoleNotFound     = errors.New("角色不存在")
)

var _ UserService = (*UserServiceImpl)(nil)

type UserView struct {
	ID        uint      `json:"id"`
	UserName  string    `json:"userName"`
	IsAdmin   bool      `json:"isAdmin"`
	RoleIDs   []int64   `json:"roleIds,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateUserInput struct {
	UserName string
	Password string
	IsAdmin  bool
	RoleIDs  []int64
}

type UpdateUserInput struct {
	UserName *string
	Password *string
	IsAdmin  *bool
	RoleIDs  *[]int64
}

type UserService interface {
	RegUser(user data.User) (err error, token string)
	Login(user data.User) (err error, token string)
	ListUsers() ([]UserView, error)
	GetUser(id int64) (UserView, error)
	CreateUser(input CreateUserInput) (UserView, error)
	UpdateUser(id int64, input UpdateUserInput) (UserView, error)
	DeleteUser(id int64) error
}

type UserServiceImpl struct {
	userRepo repo.UserRepo
	roleRepo *repo.RoleRepo
}

func NewUserServiceImpl(userRepo repo.UserRepo, roleRepos ...repo.RoleRepo) *UserServiceImpl {
	service := &UserServiceImpl{userRepo: userRepo}
	if len(roleRepos) > 0 {
		service.roleRepo = &roleRepos[0]
	}
	return service
}

func (b UserServiceImpl) Login(user data.User) (err error, token string) {
	userName := strings.TrimSpace(user.UserName)
	reUser, findErr := b.userRepo.FindByUserName(userName)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return ErrLogin, ""
		}
		log.Printf("登录失败：用户查询出错 %v", findErr)
		return ErrLogin, ""
	}

	valid, needsRehash, verifyErr := VerifyPassword(reUser.Password, user.Password)
	if verifyErr != nil || !valid {
		return ErrLogin, ""
	}
	if needsRehash {
		hash, hashErr := HashPassword(user.Password)
		if hashErr == nil {
			if updateErr := b.userRepo.UpdatePassword(int64(reUser.ID), hash); updateErr != nil {
				log.Printf("登录成功但密码重哈希失败，user_id=%d: %v", reUser.ID, updateErr)
			} else {
				reUser.Password = hash
			}
		}
	}

	roles, permissions, authErr := b.authorizationFor(reUser)
	if authErr != nil {
		log.Printf("登录成功但加载权限失败，user_id=%d: %v", reUser.ID, authErr)
		return authErr, ""
	}

	token, err = GenerateToken(reUser, roles, permissions)
	if err != nil {
		log.Printf("登录失败：生成token出错 %v", err)
	}
	return err, token
}

func (b UserServiceImpl) RegUser(user data.User) (err error, token string) {
	userName := strings.TrimSpace(user.UserName)
	if userName == "" {
		return ErrUserNameRequired, ""
	}
	if _, findErr := b.userRepo.FindByUserName(userName); findErr == nil {
		return ErrUserExists, ""
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return findErr, ""
	}

	hash, hashErr := HashPassword(user.Password)
	if hashErr != nil {
		return hashErr, ""
	}
	user.UserName = userName
	user.Password = hash
	if createErr := b.userRepo.CreateWithRoles(&user, nil); createErr != nil {
		log.Printf("用户注册失败: %v", createErr)
		return createErr, ""
	}

	roles, permissions, authErr := b.authorizationFor(user)
	if authErr != nil {
		return authErr, ""
	}
	token, err = GenerateToken(user, roles, permissions)
	return err, token
}

func (b UserServiceImpl) ListUsers() ([]UserView, error) {
	users, err := b.userRepo.List()
	if err != nil {
		return nil, err
	}
	result := make([]UserView, 0, len(users))
	for _, user := range users {
		view, viewErr := b.toUserView(user)
		if viewErr != nil {
			return nil, viewErr
		}
		result = append(result, view)
	}
	return result, nil
}

func (b UserServiceImpl) GetUser(id int64) (UserView, error) {
	user, err := b.userRepo.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserView{}, ErrUserNotFound
		}
		return UserView{}, err
	}
	return b.toUserView(user)
}

func (b UserServiceImpl) CreateUser(input CreateUserInput) (UserView, error) {
	userName := strings.TrimSpace(input.UserName)
	if userName == "" {
		return UserView{}, ErrUserNameRequired
	}
	if _, err := b.userRepo.FindByUserName(userName); err == nil {
		return UserView{}, ErrUserExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return UserView{}, err
	}

	roleIDs, err := b.normalizeRoleIDs(input.RoleIDs)
	if err != nil {
		return UserView{}, err
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		return UserView{}, err
	}

	user := data.User{
		UserName:    userName,
		Password:    hash,
		IsAdminFlag: input.IsAdmin,
	}
	if err = b.userRepo.CreateWithRoles(&user, roleIDs); err != nil {
		return UserView{}, err
	}
	return b.toUserView(user)
}

func (b UserServiceImpl) UpdateUser(id int64, input UpdateUserInput) (UserView, error) {
	current, err := b.userRepo.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserView{}, ErrUserNotFound
		}
		return UserView{}, err
	}

	fields := make(map[string]interface{})
	if input.UserName != nil {
		userName := strings.TrimSpace(*input.UserName)
		if userName == "" {
			return UserView{}, ErrUserNameRequired
		}
		if userName != current.UserName {
			if _, findErr := b.userRepo.FindByUserName(userName); findErr == nil {
				return UserView{}, ErrUserExists
			} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return UserView{}, findErr
			}
			fields["user_name"] = userName
		}
	}
	if input.Password != nil && *input.Password != "" {
		hash, hashErr := HashPassword(*input.Password)
		if hashErr != nil {
			return UserView{}, hashErr
		}
		fields["password"] = hash
	}
	if input.IsAdmin != nil {
		fields["is_admin"] = *input.IsAdmin
	}

	var roleIDs *[]int64
	if input.RoleIDs != nil {
		normalized, normalizeErr := b.normalizeRoleIDs(*input.RoleIDs)
		if normalizeErr != nil {
			return UserView{}, normalizeErr
		}
		roleIDs = &normalized
	}
	if err = b.userRepo.UpdateWithRoles(id, fields, roleIDs); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserView{}, ErrUserNotFound
		}
		return UserView{}, err
	}
	return b.GetUser(id)
}

func (b UserServiceImpl) DeleteUser(id int64) error {
	err := b.userRepo.DeleteWithRelations(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserNotFound
	}
	return err
}

func (b UserServiceImpl) authorizationFor(user data.User) ([]string, []string, error) {
	if b.roleRepo == nil {
		return []string{}, []string{}, nil
	}
	roles, err := b.roleRepo.RoleNamesByUserID(int64(user.ID))
	if err != nil {
		return nil, nil, err
	}
	permissions, err := b.roleRepo.PermissionCodesByUserID(int64(user.ID))
	if err != nil {
		return nil, nil, err
	}
	return roles, permissions, nil
}

func (b UserServiceImpl) toUserView(user data.User) (UserView, error) {
	view := UserView{
		ID:        user.ID,
		UserName:  user.UserName,
		IsAdmin:   user.IsAdmin(),
		RoleIDs:   make([]int64, 0),
		Roles:     make([]string, 0),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	if b.roleRepo == nil {
		return view, nil
	}
	roleIDs, err := b.roleRepo.RoleIDsByUserID(int64(user.ID))
	if err != nil {
		return UserView{}, err
	}
	roleNames, err := b.roleRepo.RoleNamesByUserID(int64(user.ID))
	if err != nil {
		return UserView{}, err
	}
	view.RoleIDs = roleIDs
	view.Roles = roleNames
	return view, nil
}

func (b UserServiceImpl) normalizeRoleIDs(roleIDs []int64) ([]int64, error) {
	if len(roleIDs) == 0 {
		return []int64{}, nil
	}
	if b.roleRepo == nil {
		return nil, ErrRoleNotFound
	}
	seen := make(map[int64]struct{}, len(roleIDs))
	normalized := make([]int64, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID <= 0 {
			return nil, ErrRoleNotFound
		}
		if _, ok := seen[roleID]; ok {
			continue
		}
		seen[roleID] = struct{}{}
		normalized = append(normalized, roleID)
	}
	count, err := b.roleRepo.CountByIDs(normalized)
	if err != nil {
		return nil, err
	}
	if count != int64(len(normalized)) {
		return nil, ErrRoleNotFound
	}
	return normalized, nil
}
