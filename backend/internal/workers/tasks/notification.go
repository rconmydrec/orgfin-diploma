package tasks

import (
	"fmt"
	"time"
)

// NotificationConfig carries admin notification settings into task handlers.
// It avoids passing the entire config.Config into task handlers and keeps
// dependencies minimal and testable.
type NotificationConfig struct {
	AdminEmails []string
	Environment string
}

// buildExchangeRateNotificationSubject generates the email subject for the exchange rate update notification.
func buildExchangeRateNotificationSubject(environment string) string {
	return fmt.Sprintf("[%s] Currency rates updated", environment)
}

// buildExchangeRateNotificationBody generates the HTML body for the exchange rate update notification email.
func buildExchangeRateNotificationBody(environment, updatedAt string, rateCount int, baseCurrency string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Currency Rate Update Notification</title>
    <style>
        body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 0; }
        .container { max-width: 600px; margin: 40px auto; background: #ffffff; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); padding: 30px; }
        h1 { color: #333333; font-size: 22px; }
        p { color: #666666; line-height: 1.6; }
        .details { margin: 20px 0; }
        .details p { margin: 4px 0; }
        .footer { text-align: center; margin-top: 30px; font-size: 12px; color: #aaaaaa; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Currency Rate Update Notification</h1>
        <p>Dear Team,</p>
        <p>This is to inform you that the currency rates have been successfully updated in the <strong>` + environment + `</strong> environment.</p>
        <div class="details">
            <p><strong>Time:</strong> ` + updatedAt + `</p>
            <p><strong>Rates updated:</strong> ` + fmt.Sprintf("%d", rateCount) + `</p>
            <p><strong>Base currency:</strong> ` + baseCurrency + `</p>
            <p><strong>Project:</strong> Orgfin</p>
        </div>
        <p>If you have any questions or need further assistance, please contact the support team.</p>
        <p>Best regards,<br>The Orgfin Team</p>
        <div class="footer">
            <p>&copy; ` + fmt.Sprintf("%d", time.Now().Year()) + ` Orgfin. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
}
