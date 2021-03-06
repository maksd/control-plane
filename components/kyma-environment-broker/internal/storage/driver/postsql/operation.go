package postsql

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/storage"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/pivotal-cf/brokerapi/v7/domain"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession"
	"github.com/pkg/errors"
)

type operations struct {
	dbsession.Factory
}

func NewOperation(sess dbsession.Factory) *operations {
	return &operations{
		Factory: sess,
	}
}

// InsertProvisioningOperation insert new ProvisioningOperation to storage
func (s *operations) InsertProvisioningOperation(operation internal.ProvisioningOperation) error {
	session := s.NewWriteSession()
	dto, err := provisioningOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while inserting provisioning operation (id: %s)", operation.ID)
	}
	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Warn(errors.Wrap(err, "while insert operation"))
			return false, nil
		}
		return true, nil
	})
	return lastErr
}

// GetProvisioningOperationByID fetches the ProvisioningOperation by given ID, returns error if not found
func (s *operations) GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage"))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := toProvisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetProvisioningOperationByInstanceID fetches the ProvisioningOperation by given instanceID, returns error if not found
func (s *operations) GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(instanceID, dbmodel.OperationTypeProvision)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := toProvisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateProvisioningOperation updates ProvisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateProvisioningOperation(op internal.ProvisioningOperation) (*internal.ProvisioningOperation, error) {
	session := s.NewWriteSession()
	op.UpdatedAt = time.Now()
	dto, err := provisioningOperationToDTO(&op)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(op.ID)
			if lastErr != nil {
				log.Warn(errors.Wrapf(lastErr, "while getting Operation").Error())
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", op.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	op.Version = op.Version + 1
	return &op, lastErr
}

// InsertDeprovisioningOperation insert new DeprovisioningOperation to storage
func (s *operations) InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error {
	session := s.NewWriteSession()

	dto, err := deprovisioningOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Warn(errors.Wrap(err, "while insert operation"))
			return false, nil
		}
		return true, nil
	})
	return lastErr
}

// GetDeprovisioningOperationByID fetches the DeprovisioningOperation by given ID, returns error if not found
func (s *operations) GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage"))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := toDeprovisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetDeprovisioningOperationByInstanceID fetches the DeprovisioningOperation by given instanceID, returns error if not found
func (s *operations) GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(instanceID, dbmodel.OperationTypeDeprovision)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := toDeprovisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateDeprovisioningOperation updates DeprovisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateDeprovisioningOperation(operation internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()

	dto, err := deprovisioningOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(operation.ID)
			if lastErr != nil {
				log.Warn(errors.Wrapf(lastErr, "while getting Operation").Error())
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// InsertUpgradeKymaOperation insert new UpgradeKymaOperation to storage
func (s *operations) InsertUpgradeKymaOperation(operation internal.UpgradeKymaOperation) error {
	session := s.NewWriteSession()
	dto, err := upgradeKymaOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while inserting upgrade kyma operation (id: %s)", operation.Operation.ID)
	}
	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Warn(errors.Wrap(err, "while insert operation"))
			return false, nil
		}

		//todo - insert link to orchestration
		return true, nil
	})
	return lastErr
}

// GetUpgradeKymaOperationByID fetches the UpgradeKymaOperation by given ID, returns error if not found
func (s *operations) GetUpgradeKymaOperationByID(operationID string) (*internal.UpgradeKymaOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage"))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := toUpgradeKymaOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetUpgradeKymaOperationByInstanceID fetches the UpgradeKymaOperation by given instanceID, returns error if not found
func (s *operations) GetUpgradeKymaOperationByInstanceID(instanceID string) (*internal.UpgradeKymaOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(instanceID, dbmodel.OperationTypeUpgradeKyma)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := toUpgradeKymaOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

func (s *operations) ListUpgradeKymaOperationsByInstanceID(instanceID string) ([]internal.UpgradeKymaOperation, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceID, dbmodel.OperationTypeUpgradeKyma)
		if lastErr != nil {
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := toUpgradeKymaOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateUpgradeKymaOperation updates UpgradeKymaOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateUpgradeKymaOperation(operation internal.UpgradeKymaOperation) (*internal.UpgradeKymaOperation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := upgradeKymaOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(operation.Operation.ID)
			if lastErr != nil {
				log.Warn(errors.Wrapf(lastErr, "while getting Operation").Error())
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.Operation.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// GetOperationByID returns Operation with given ID. Returns an error if the operation does not exists.
func (s *operations) GetOperationByID(operationID string) (*internal.Operation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Warn(errors.Wrapf(lastErr, "while reading Operation from the storage").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	op := toOperation(&operation)
	return &op, nil
}

func (s *operations) GetOperationsInProgressByType(operationType dbmodel.OperationType) ([]internal.Operation, error) {
	session := s.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		dto, err := session.GetOperationsInProgressByType(operationType)
		if err != nil {
			log.Warn(errors.Wrapf(err, "while getting Operations from the storage").Error())
			return false, nil
		}
		operations = dto
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return toOperations(operations), nil
}

func (s *operations) GetOperationStats() (internal.OperationStats, error) {
	entries, err := s.NewReadSession().GetOperationStats()
	if err != nil {
		return internal.OperationStats{}, err
	}

	result := internal.OperationStats{
		Provisioning:   make(map[domain.LastOperationState]int),
		Deprovisioning: make(map[domain.LastOperationState]int),
	}
	for _, e := range entries {
		switch dbmodel.OperationType(e.Type) {
		case dbmodel.OperationTypeProvision:
			result.Provisioning[domain.LastOperationState(e.State)] = e.Total
		case dbmodel.OperationTypeDeprovision:
			result.Deprovisioning[domain.LastOperationState(e.State)] = e.Total
		}
	}
	return result, nil
}

func (s *operations) GetOperationStatsForOrchestration(orchestrationID string) (map[string]int, error) {
	entries, err := s.NewReadSession().GetOperationStatsForOrchestration(orchestrationID)
	if err != nil {
		return map[string]int{}, err
	}
	result := make(map[string]int, 5)
	for _, entry := range entries {
		result[entry.State] = entry.Total
	}
	return result, nil
}

func (s *operations) GetOperationsForIDs(operationIDList []string) ([]internal.Operation, error) {
	session := s.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		dto, err := session.GetOperationsForIDs(operationIDList)
		if err != nil {
			log.Warn(errors.Wrapf(err, "while getting Operations from the storage").Error())
			return false, nil
		}
		operations = dto
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return toOperations(operations), nil
}

func (s *operations) ListUpgradeKymaOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.UpgradeKymaOperation, int, int, error) {
	session := s.NewReadSession()
	var (
		operations        = make([]dbmodel.OperationDTO, 0)
		lastErr           error
		count, totalCount int
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, count, totalCount, lastErr = session.ListOperationsByOrchestrationID(orchestrationID, filter)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operations for orchestration ID %s not exist", orchestrationID)
				return false, lastErr
			}
			log.Errorf("while reading Operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while getting operation by ID: %v", lastErr)
	}
	ret, err := toUpgradeKymaOperationList(operations)
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, count, totalCount, nil
}

func toOperation(op *dbmodel.OperationDTO) internal.Operation {
	return internal.Operation{
		ID:                     op.ID,
		CreatedAt:              op.CreatedAt,
		UpdatedAt:              op.UpdatedAt,
		ProvisionerOperationID: op.TargetOperationID,
		State:                  domain.LastOperationState(op.State),
		InstanceID:             op.InstanceID,
		Description:            op.Description,
		Version:                op.Version,
		OrchestrationID:        storage.SQLNullStringToString(op.OrchestrationID),
	}
}

func toOperations(op []dbmodel.OperationDTO) []internal.Operation {
	operations := make([]internal.Operation, 0)
	for _, o := range op {
		operations = append(operations, toOperation(&o))
	}
	return operations
}

func toProvisioningOperation(op *dbmodel.OperationDTO) (*internal.ProvisioningOperation, error) {
	if op.Type != dbmodel.OperationTypeProvision {
		return nil, errors.New(fmt.Sprintf("expected operation type Provisioning, but was %s", op.Type))
	}
	var operation internal.ProvisioningOperation
	err := json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation = toOperation(op)

	return &operation, nil
}

func provisioningOperationToDTO(op *internal.ProvisioningOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing provisioning data %v", op)
	}

	ret := operationToDB(&op.Operation)
	ret.Data = string(serialized)
	ret.Type = dbmodel.OperationTypeProvision
	return ret, nil
}

func toDeprovisioningOperation(op *dbmodel.OperationDTO) (*internal.DeprovisioningOperation, error) {
	if op.Type != dbmodel.OperationTypeDeprovision {
		return nil, errors.New(fmt.Sprintf("expected operation type Provisioning, but was %s", op.Type))
	}
	var operation internal.DeprovisioningOperation
	err := json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation = toOperation(op)

	return &operation, nil
}

func deprovisioningOperationToDTO(op *internal.DeprovisioningOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing deprovisioning data %v", op)
	}

	ret := operationToDB(&op.Operation)
	ret.Data = string(serialized)
	ret.Type = dbmodel.OperationTypeDeprovision
	return ret, nil
}

func toUpgradeKymaOperation(op *dbmodel.OperationDTO) (*internal.UpgradeKymaOperation, error) {
	if op.Type != dbmodel.OperationTypeUpgradeKyma {
		return nil, errors.New(fmt.Sprintf("expected operation type Upgrade Kyma, but was %s", op.Type))
	}
	var operation internal.UpgradeKymaOperation
	err := json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation = toOperation(op)
	operation.RuntimeOperation.ID = op.ID
	if op.OrchestrationID.Valid {
		operation.OrchestrationID = op.OrchestrationID.String
	}

	return &operation, nil
}

func toUpgradeKymaOperationList(ops []dbmodel.OperationDTO) ([]internal.UpgradeKymaOperation, error) {
	result := make([]internal.UpgradeKymaOperation, 0)

	for _, op := range ops {
		o, err := toUpgradeKymaOperation(&op)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade kyma operation")
		}
		result = append(result, *o)
	}

	return result, nil
}

func upgradeKymaOperationToDTO(op *internal.UpgradeKymaOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing provisioning data %v", op)
	}

	ret := operationToDB(&op.Operation)
	ret.Data = string(serialized)
	ret.Type = dbmodel.OperationTypeUpgradeKyma
	ret.OrchestrationID = storage.StringToSQLNullString(op.OrchestrationID)
	return ret, nil
}

func operationToDB(op *internal.Operation) dbmodel.OperationDTO {
	return dbmodel.OperationDTO{
		ID:                op.ID,
		TargetOperationID: op.ProvisionerOperationID,
		State:             string(op.State),
		Description:       op.Description,
		UpdatedAt:         op.UpdatedAt,
		CreatedAt:         op.CreatedAt,
		Version:           op.Version,
		InstanceID:        op.InstanceID,
		OrchestrationID:   storage.StringToSQLNullString(op.OrchestrationID),
	}
}
