package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/AutomaticOrca/simplebank/db/mock"
	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/mail"
	mockmail "github.com/AutomaticOrca/simplebank/mail/mock"
	"github.com/AutomaticOrca/simplebank/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// randomUserForTest 生成用于测试的随机用户信息和明文密码
func randomUserForTest(t *testing.T) (user db.User, password string) {
	password = util.RandomString(8)
	user = db.User{
		Username: util.RandomOwner(),
		FullName: util.RandomOwner(),
		Email:    util.RandomEmail(),
		// HashedPassword 将在 API handler 内部通过 util.HashPassword 生成
		// IsEmailVerified 默认为 false
		// PasswordChangedAt 和 CreatedAt 由数据库生成，我们将在 mock 返回时模拟它们
	}
	return
}

// newTestServer 是一个辅助函数，用于创建带有 mock 依赖的 api.Server 实例
// 这样我们可以在不同的测试用例中复用服务器的创建逻辑
func newTestServer(t *testing.T, store db.Store, mailer mail.EmailSender) *Server {
	// 为测试创建一个最小化但足够使用的配置
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),     // PasetoMaker 创建时需要
		AccessTokenDuration: time.Minute,               // TokenMaker 创建时需要
		FrontendBaseURL:     "http://test.example.com", // sendVerificationEmailAsync 中会用到
		// 根据你的 NewServer 函数和被测 handler 的实际需求，添加其他必要的配置字段
		// 例如，如果 NewServer 或 setupRouter 中用到了其他 config 值，也需要在这里提供
	}

	// 调用你 api 包中的 NewServer 函数，传入 mock 的 store 和 mailer
	server, err := NewServer(config, store, mailer)
	require.NoError(t, err) // 确保服务器实例创建成功
	return server
}

// eqCreateUserTxParamsMatcher 是一个 gomock.Matcher，用于精确比较 db.CreateUserTxParams
// 它特别处理 HashedPassword (通过比较原始密码) 和 AfterCreate 回调
type eqCreateUserTxParamsMatcher struct {
	expectedParams db.CreateUserParams // 我们期望 CreateUserParams 中的核心字段是什么
	password       string              // 用于哈希后比较的原始密码
}

// Matches 是 gomock.Matcher 接口的核心方法
func (m eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	actualArg, ok := x.(db.CreateUserTxParams) // 类型断言，看传入的是不是 db.CreateUserTxParams
	if !ok {
		return false // 类型不匹配，肯定不相等
	}

	// 1. 验证密码：检查传入的明文密码 m.password 是否能哈希匹配到实际参数中的 actualArg.HashedPassword
	err := util.CheckPassword(m.password, actualArg.HashedPassword)
	if err != nil {
		return false // 密码不匹配
	}

	// 2. 比较 CreateUserParams 中的其他核心字段
	if m.expectedParams.Username != actualArg.Username ||
		m.expectedParams.FullName != actualArg.FullName ||
		m.expectedParams.Email != actualArg.Email {
		return false // 核心字段不匹配
	}

	// 3. 验证 AfterCreate 回调
	if actualArg.AfterCreate == nil {
		return false // AfterCreate 回调函数不应该为 nil
	}
	// 在单元测试中，我们主要关心 AfterCreate 被正确设置了，并且它本身（同步部分）被调用时不会出错。
	// 它内部启动的 goroutine 的行为，应该由 sendVerificationEmailAsync 方法的独立单元测试来覆盖。
	// 构造一个符合 AfterCreate 函数签名的虚拟 user 对象来调用它。
	dummyUserForCallback := db.User{
		Username: actualArg.Username, // 使用从实际参数中获取的值
		FullName: actualArg.FullName,
		Email:    actualArg.Email,
		// 其他字段对于测试 AfterCreate 能否被调用且不返回 error 可能不那么重要
	}
	if err := actualArg.AfterCreate(dummyUserForCallback); err != nil {
		return false // AfterCreate 回调本身返回了错误 (我们的实现中它返回 nil)
	}

	return true // 所有检查都通过了
}

// String 是 gomock.Matcher 接口的方法，用于在测试失败时打印有意义的信息
func (m eqCreateUserTxParamsMatcher) String() string {
	return fmt.Sprintf("matches CreateUserParams for user %s and password %s", m.expectedParams.Username, "[REDACTED_PASSWORD]")
}

// EqCreateUserTxParams 是一个工厂函数，方便创建 eqCreateUserTxParamsMatcher 实例
func EqCreateUserTxParams(params db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserTxParamsMatcher{
		expectedParams: params,
		password:       password,
	}
}

// requireBodyMatchUser 用于比较 HTTP 响应体中的 User 数据与期望的 User 数据
func requireBodyMatchUser(t *testing.T, body io.Reader, expectedInputUser db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUserResp userResponse // 这是你的 api.userResponse 结构体
	err = json.Unmarshal(data, &gotUserResp)
	require.NoError(t, err, "Error unmarshalling response body: %s", string(data))

	// 比较那些应该由客户端输入决定或与输入相关的字段
	require.Equal(t, expectedInputUser.Username, gotUserResp.Username)
	require.Equal(t, expectedInputUser.FullName, gotUserResp.FullName)
	require.Equal(t, expectedInputUser.Email, gotUserResp.Email)

	// 对于由服务器（或数据库）生成的时间戳，我们检查它们是否是非零值
	// 这表明它们在响应中被正确地设置了
	require.False(t, gotUserResp.PasswordChangedAt.IsZero(), "PasswordChangedAt from response should not be a zero value")
	require.False(t, gotUserResp.CreatedAt.IsZero(), "CreatedAt from response should not be a zero value")
}

func TestCreateUserAPI(t *testing.T) {
	// 为所有测试用例生成一组基础的随机用户数据
	// 'user' 主要用于构造请求体和在 checkResponse 中比较核心字段
	// 'password' 是对应的明文密码
	user, password := randomUserForTest(t)

	testCases := []struct {
		name          string                                                          // 测试用例的名称
		body          gin.H                                                           // 用于构造请求 JSON body
		buildStubs    func(store *mockdb.MockStore, mailer *mockmail.MockEmailSender) // 用于设置 mock 的期望
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)         // 用于检查 HTTP 响应
	}{
		// --- 测试用例 1: 成功创建用户 (OK scenario) ---
		{
			name: "OK",
			body: gin.H{ // 构造一个有效的请求体
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, mailer *mockmail.MockEmailSender) {
				// 准备 CreateUserTx 方法期望接收的参数 (CreateUserParams 部分)
				argParamsForMatcher := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					Email:    user.Email,
					// HashedPassword 由 handler 内部生成并传递给 CreateUserTx，
					// 我们的 EqCreateUserTxParams matcher 会用明文 password 来校验它。
				}

				// 准备一个模拟的、当 CreateUserTx 成功时从数据库返回的 User 对象
				// 这个对象应该包含由数据库（或我们的 mock）设置的时间戳
				userReturnedByTx := user                        // 从 randomUserForTest 获取基础信息
				userReturnedByTx.CreatedAt = time.Now()         // 模拟数据库填充此字段
				userReturnedByTx.PasswordChangedAt = time.Now() // 模拟数据库填充此字段
				// IsEmailVerified 默认为 false，通常在验证邮件后更新

				// 设置对 store.CreateUserTx 的期望
				store.EXPECT().
					CreateUserTx(
						gomock.Any(), // context 参数，通常用 gomock.Any()
						EqCreateUserTxParams(argParamsForMatcher, password), // 使用自定义 matcher 验证参数
					).
					Times(1).                                                  // 期望被调用一次
					Return(db.CreateUserTxResult{User: userReturnedByTx}, nil) // 模拟成功，并返回构造好的 User 对象

				// 添加对 CreateVerifyEmail 的期望
				store.EXPECT().
					CreateVerifyEmail(
						gomock.Any(),
						gomock.Any(),
					).
					Times(1).
					Return(db.VerifyEmail{
						ID:         1,
						Username:   user.Username,
						Email:      user.Email,
						SecretCode: "test-secret-code",
						IsUsed:     false,
						CreatedAt:  time.Now(),
						ExpiredAt:  time.Now().Add(15 * time.Minute),
					}, nil)

				// 添加对 SendEmail 的期望
				mailer.EXPECT().
					SendEmail(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code) // 期望 HTTP 状态码为 201 Created
				// 使用 requireBodyMatchUser 来验证响应体中的核心字段和时间戳的非零性
				// 这里的 'user' 是最初 randomUserForTest 生成的，用于比较那些不应被服务器改变的输入字段
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		// --- 接下来可以添加其他测试用例，例如参数验证失败、数据库错误等 ---
		// 例如：无效邮箱格式
		{
			name: "InvalidEmail",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     "invalid-email", // 无效的邮箱格式
			},
			buildStubs: func(store *mockdb.MockStore, mailer *mockmail.MockEmailSender) {
				// 如果输入验证失败，CreateUserTx 不应该被调用
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		// 例如：密码太短
		{
			name: "PasswordTooShort",
			body: gin.H{
				"username":  user.Username,
				"password":  "123", // 密码少于6位
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, mailer *mockmail.MockEmailSender) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		// 例如：数据库返回唯一性冲突错误
		{
			name: "DuplicateUsernameOrEmail",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, mailer *mockmail.MockEmailSender) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()). // 不关心具体参数，因为期望错误
					Times(1).
					Return(db.CreateUserTxResult{}, &pq.Error{Code: "23505"}) // 模拟 PostgreSQL 唯一约束冲突错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		// 例如：数据库返回通用内部错误
		{
			name: "InternalServerErrorFromDB",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, mailer *mockmail.MockEmailSender) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateUserTxResult{}, sql.ErrConnDone) // 模拟一个通用数据库连接错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	gin.SetMode(gin.TestMode) // 将 Gin 设置为测试模式，以减少不必要的控制台日志输出

	// 循环执行所有测试用例
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			// 为每个测试用例创建一个新的 gomock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish() // 在子测试结束时，验证所有 mock 期望是否都已满足

			// 创建 mock 实例
			storeMock := mockdb.NewMockStore(ctrl)
			mailerMock := mockmail.NewMockEmailSender(ctrl)

			// 调用 buildStubs 函数来设置当前测试用例的 mock 期望
			if tc.buildStubs != nil { // 确保 buildStubs 不是 nil
				tc.buildStubs(storeMock, mailerMock)
			}

			// 创建测试服务器实例，注入 mock 依赖
			server := newTestServer(t, storeMock, mailerMock)

			// 创建一个 HTTP 响应记录器
			recorder := httptest.NewRecorder()

			// 将请求体（gin.H map）序列化为 JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			// 创建一个 HTTP POST 请求
			request, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(data))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json") // 设置请求头

			// 让 Gin 路由处理这个请求，并将响应写入 recorder
			server.router.ServeHTTP(recorder, request)

			// 调用 checkResponse 函数来验证 HTTP 响应的状态码和内容
			tc.checkResponse(t, recorder)

			// 等待一小段时间，确保异步操作完成
			time.Sleep(200 * time.Millisecond)
		})
	}
}
