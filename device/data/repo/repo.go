package repo

type baseRepo interface {
}

var _ baseRepo = (*baseRepoImpl)(nil)

type baseRepoImpl struct {
}
