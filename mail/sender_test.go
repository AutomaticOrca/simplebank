package mail

import (
	"testing"

	"github.com/AutomaticOrca/simplebank/util"
	"github.com/stretchr/testify/require"
)

func TestSendEmailWithGmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	config, err := util.LoadConfig()
	require.NoError(t, err, "Failed to load config. Ensure .env is accessible from the test's working directory or environment variables are set.")

	sender := NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)

	subject := "A test email"
	content := `
	<h1>Hello from JY Liang</h1>
	<p>This is a test message from <a href="https://simplebank-frontend.vercel.app/">JY Bank</a></p>
	`
	to := []string{"liangjiaying1013@gmail.com"}
	attachFiles := []string{"./README.md"}

	err = sender.SendEmail(subject, content, to, nil, nil, attachFiles)
	require.NoError(t, err)
}
