package storage

import (
	"context"
	"encoding/json"
	"log"

	"cloud.google.com/go/datastore"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type TaskRunStorage struct {
	client *datastore.Client
}

type entity struct {
	JSON []byte `datastore:"json,noindex"`
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

func (s *TaskRunStorage) Exists(ctx context.Context, name string) bool {
	// TODO: key should  be (name+namespace), or maybe UID?
	k := datastore.NameKey("TaskRun", name, nil)
	err := s.client.Get(ctx, k, &v1beta1.TaskRun{})
	if err == nil {
		return true
	}
	if err == datastore.ErrNoSuchEntity {
		return false
	}
	// err != nil
	log.Printf("error getting TaskRun %q: %v", name, err)
	return false
}

func (s *TaskRunStorage) Insert(ctx context.Context, tr *v1beta1.TaskRun) error {
	// TODO: key should  be (name+namespace), or maybe UID?
	k := datastore.NameKey("TaskRun", tr.Name, nil)
	b, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	_, err = s.client.Put(ctx, k, &entity{JSON: b})
	return err
}

func (s *TaskRunStorage) Update(ctx context.Context, name string, trs v1beta1.TaskRunStatus) (*v1beta1.TaskRun, error) {
	// TODO: key should  be (name+namespace), or maybe UID?
	k := datastore.NameKey("TaskRun", name, nil)
	var tr v1beta1.TaskRun
	if _, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var e entity
		if err := tx.Get(k, &e); err != nil {
			return err
		}

		var tr v1beta1.TaskRun
		if err := json.Unmarshal(e.JSON, &tr); err != nil {
			return err
		}

		tr.Status = trs

		b, err := json.Marshal(tr)
		if err != nil {
			return err
		}
		tx.Mutate(datastore.NewUpdate(k, entity{JSON: b}))
		return nil
	}); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (s *TaskRunStorage) Get(ctx context.Context, name string) (*v1beta1.TaskRun, error) {
	k := datastore.NameKey("TaskRun", name, nil)
	var e entity
	if err := s.client.Get(ctx, k, &e); err != nil {
		return nil, err
	}
	var tr v1beta1.TaskRun
	if err := json.Unmarshal(e.JSON, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (s *TaskRunStorage) Delete(ctx context.Context, name string) error {
	k := datastore.NameKey("TaskRun", name, nil)
	err := s.client.Delete(ctx, k)
	return err
}
