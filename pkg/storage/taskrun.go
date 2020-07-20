package storage

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type TaskRunStorage struct {
	client *datastore.Client
}

func NewTaskRunStorage(ctx context.Context, projectID string) (*TaskRunStorage, error) {
	c, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &TaskRunStorage{
		client: c,
	}, nil
}

func (s *TaskRunStorage) Upsert(ctx context.Context, tr *v1beta1.TaskRun) error {
	// TODO: key should  be (name+namespace), or maybe UID?
	k := datastore.NameKey("TaskRun", tr.Name, nil)
	_, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		tx.Mutate(datastore.NewUpsert(k, tr))
		_, err := tx.Commit()
		return err
	})
	return err
}
