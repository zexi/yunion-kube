package data

import (
	"k8s.io/helm/pkg/repo"
)

type IRepoBackend interface {
	Add(repo *repo.Entry) error
	Delete(repoName string) error
	Modify(repoName string, newRepo *repo.Entry) error
	Show(repoName string) (*repo.Entry, error)
	Update(repoName string) error
}
