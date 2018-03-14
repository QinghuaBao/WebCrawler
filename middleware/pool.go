package middleware

import (
	"reflect"
	"fmt"
	"sync"
	"errors"
)

/**
 * Created by bqh on 2018/3/9.
 * E-mail:M201672845@hust.edu.cn
 */

type Entity interface {
	Id() uint32
}

type Pool interface {
	Take() (Entity, error)
	Return(entity Entity) error
	Total() uint32
	Used() uint32
}


// 创建实体池。
func NewPool(total uint32, entityType reflect.Type, genEntity func() Entity) (Pool, error) {
	if total == 0 {
		errMsg :=
			fmt.Sprintf("The pool can not be initialized! (total=%d)\n", total)
		return nil, errors.New(errMsg)
	}
	size := int(total)
	container := make(chan Entity, size)
	idContainer := make(map[uint32]bool)
	for i := 0; i < size; i++ {
		newEntity := genEntity()
		if entityType != reflect.TypeOf(newEntity) {
			errMsg :=
				fmt.Sprintf("The type of result of function genEntity() is NOT %s!\n", entityType)
			return nil, errors.New(errMsg)
		}
		container <- newEntity
		idContainer[newEntity.Id()] = true
	}
	pool := &myPool{
		total:       total,
		etype:       entityType,
		genEntity:   genEntity,
		container:   container,
		idContainer: idContainer,
	}
	return pool, nil
}

// 实体池的实现类型。
type myPool struct {
	total       uint32          // 池的总容量。
	etype       reflect.Type    // 池中实体的类型。
	genEntity   func() Entity   // 池中实体的生成函数。
	container   chan Entity     // 实体容器。
	idContainer map[uint32]bool // 实体ID的容器。
	mutex       sync.Mutex      // 针对实体ID容器操作的互斥锁。
}

func (pool *myPool) Take() (Entity, error) {
	entity, ok := <-pool.container
	if !ok {
		return nil, errors.New("The inner container is invalid!")
	}
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	pool.idContainer[entity.Id()] = false
	return entity, nil
}

func (pool *myPool) Return(entity Entity) error {
	if entity == nil {
		return errors.New("The returning entity is invalid!")
	}
	if pool.etype != reflect.TypeOf(entity) {
		errMsg := fmt.Sprintf("The type of returning entity is NOT %s!\n", pool.etype)
		return errors.New(errMsg)
	}
	entityId := entity.Id()
	casResult := pool.compareAndSetForIdContainer(entityId, false, true)
	if casResult == 1 {
		pool.container <- entity
		return nil
	} else if casResult == 0 {
		errMsg := fmt.Sprintf("The entity (id=%d) is already in the pool!\n", entityId)
		return errors.New(errMsg)
	} else {
		errMsg := fmt.Sprintf("The entity (id=%d) is illegal!\n", entityId)
		return errors.New(errMsg)
	}
}

// 比较并设置实体ID容器中与给定实体ID对应的键值对的元素值。
// 结果值：
//       -1：表示键值对不存在。
//        0：表示操作失败。
//        1：表示操作成功。
func (pool *myPool) compareAndSetForIdContainer(
	entityId uint32, oldValue bool, newValue bool) int8 {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	v, ok := pool.idContainer[entityId]
	if !ok {
		return -1
	}
	if v != oldValue {
		return 0
	}
	pool.idContainer[entityId] = newValue
	return 1
}

func (pool *myPool) Total() uint32 {
	return pool.total
}

func (pool *myPool) Used() uint32 {
	return pool.total - uint32(len(pool.container))
}
