package lib

import (
	"errors"
	"sync"
)

// Extending Transaction to support more operations
type Transaction struct {
	operations []func() error
	mu         sync.Mutex
}

// TransactionManager extends transactions to support lists, sets, hashes, and zsets
type TransactionManager struct {
	transactions map[uint32]*Transaction
	mu           sync.Mutex
}

func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		transactions: make(map[uint32]*Transaction),
	}
}
func (tm *TransactionManager) Begin(txID uint32) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.transactions[txID] = &Transaction{
		operations: make([]func() error, 0),
	}
}

// Add operation to a transaction
func (tm *TransactionManager) AddOperation(txID uint32, operation func() error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tx, exists := tm.transactions[txID]; exists {
		tx.mu.Lock()
		defer tx.mu.Unlock()
		tx.operations = append(tx.operations, operation)
		return nil
	}
	return errors.New("transaction not found")
}
// Add List Operation to Transaction
func (tm *TransactionManager) AddListOperation(txID uint32, listOp func() error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tx, exists := tm.transactions[txID]; exists {
		tx.mu.Lock()
		defer tx.mu.Unlock()
		tx.operations = append(tx.operations, listOp)
		return nil
	}
	return errors.New("transaction not found")
}
// Add Set Operation to Transaction
func (tm *TransactionManager) AddSetOperation(txID uint32, setOp func() error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tx, exists := tm.transactions[txID]; exists {
		tx.mu.Lock()
		defer tx.mu.Unlock()
		tx.operations = append(tx.operations, setOp)
		return nil
	}
	return errors.New("transaction not found")
}
// Commit a transaction
func (tm *TransactionManager) Commit(txID uint32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tx, exists := tm.transactions[txID]; exists {
		for _, op := range tx.operations {
			if err := op(); err != nil {
				return err
			}
		}
		delete(tm.transactions, txID)
		return nil
	}
	return errors.New("transaction not found")
}

// Rollback a transaction
func (tm *TransactionManager) Rollback(txID uint32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, exists := tm.transactions[txID]; exists {
		delete(tm.transactions, txID)
		return nil
	}
	return errors.New("transaction not found")
}
