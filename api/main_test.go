package api

import (
	"github.com/AutomaticOrca/simplebank/worker/mock"
	"os"
	"testing"
	"time"

	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/util"
	"github.com/AutomaticOrca/simplebank/worker"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// create a new server for test
func newTestServer(t *testing.T, store db.Store, taskDistributor worker.TaskDistributor) *Server {
	config := util.Config{
		TokenSymmetricKey:    util.RandomString(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
	}

	ctrl := gomock.NewController(t)
	mockTaskDistributor := mock.NewMockTaskDistributor(ctrl)
	server, err := NewServer(config, store, mockTaskDistributor)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
