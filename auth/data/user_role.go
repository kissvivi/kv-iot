package data

type UserRole struct {
	UserID int64 `json:"user_id" gorm:"column:user_id"` //用户ID
	RoleID int64 `json:"role_id" gorm:"column:role_id"` //角色ID
}

func (ur *UserRole) TableName() string {
	return "user_role"
}
