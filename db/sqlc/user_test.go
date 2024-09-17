package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {

	userArg := CreateUserParams{
		Username:       "tom",                
		HashedPassword: "hashed_password123", 
		FullName:       "Tom Hanks",          
		Email:          "tom@example.com",    
	}


	user, err := testQueries.CreateUser(context.Background(), userArg)
	require.NoError(t, err)      
	require.NotEmpty(t, user)     


	require.Equal(t, userArg.Username, user.Username)
	require.Equal(t, userArg.HashedPassword, user.HashedPassword)
	require.Equal(t, userArg.FullName, user.FullName)
	require.Equal(t, userArg.Email, user.Email)

	require.NotZero(t, user.CreatedAt)   
}