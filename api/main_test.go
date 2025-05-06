package api

import (
	"os"
	"testing"
	"time"

	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/util"
	mockwk "github.com/AutomaticOrca/simplebank/worker/mock"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// create a new server for test
func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	ctrl := gomock.NewController(t)
	mockTaskDistributor := mockwk.NewMockTaskDistributor(ctrl)
	server, err := NewServer(config, store, mockTaskDistributor)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
