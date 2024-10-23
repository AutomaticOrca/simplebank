package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTxDeadlock(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)
	n := 10             // the number of concurrent transactions
	amount := int64(10) // transfer amount

	// Create channels to receive errors and results from each transaction
	errs := make(chan error)

	for i := 0; i < n; i++ {

		go func() {
			fromAccountID := account1.ID
			toAccountID := account2.ID

			if i%2 == 1 {
				fromAccountID = account2.ID
				toAccountID = account1.ID
			}
			// Execute the money transfer transaction
			_, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			// Send the error and result back through the channels
			errs <- err
		}()
	}
	// Loop through the results of each transaction
	for i := 0; i < n; i++ {
		// Check for errors in the transaction and assert no error occurred
		err := <-errs
		require.NoError(t, err)
	}

	// check the final updated balance
	updatedAccount1, err := store.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := store.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)

	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}
