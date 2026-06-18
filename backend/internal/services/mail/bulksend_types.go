package mail

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BulkProviderMeta describes a supported outbound email provider.
type BulkProviderMeta struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	NameEN      string   `json:"name_en"`
	Description string   `json:"description"`
	Fields      []string `json:"fields"`
}

// BulkProviderCatalog lists mainstream transactional / bulk email platforms.
var BulkProviderCatalog = []BulkProviderMeta{
	{Type: "local", Name: "本机 Postfix（自有域名）", NameEN: "Local Postfix (own domain)", Description: "通过服务器已安装的 Postfix 发送，需配置 SPF/DKIM/MX", Fields: []string{}},
	{Type: "custom_smtp", Name: "自定义 SMTP", NameEN: "Custom SMTP", Description: "任意 SMTP 服务器（企业邮箱、QQ/163 企业邮等）", Fields: []string{"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_tls"}},
	{Type: "sendgrid", Name: "SendGrid", NameEN: "SendGrid", Description: "Twilio SendGrid 事务邮件", Fields: []string{"api_key"}},
	{Type: "mailgun", Name: "Mailgun", NameEN: "Mailgun", Description: "Mailgun 邮件 API", Fields: []string{"api_key", "domain", "region"}},
	{Type: "amazon_ses", Name: "Amazon SES", NameEN: "Amazon SES", Description: "AWS Simple Email Service", Fields: []string{"access_key", "secret_key", "region"}},
	{Type: "postmark", Name: "Postmark", NameEN: "Postmark", Description: "Postmark 事务邮件", Fields: []string{"server_token"}},
	{Type: "brevo", Name: "Brevo (Sendinblue)", NameEN: "Brevo", Description: "Brevo 邮件营销与事务邮件", Fields: []string{"api_key"}},
	{Type: "mailjet", Name: "Mailjet", NameEN: "Mailjet", Description: "Mailjet 邮件 API", Fields: []string{"api_key", "api_secret"}},
	{Type: "sparkpost", Name: "SparkPost", NameEN: "SparkPost", Description: "Bird SparkPost 邮件投递", Fields: []string{"api_key"}},
	{Type: "resend", Name: "Resend", NameEN: "Resend", Description: "Resend 开发者邮件 API", Fields: []string{"api_key"}},
	{Type: "smtp2go", Name: "SMTP2GO", NameEN: "SMTP2GO", Description: "SMTP2GO 中继服务", Fields: []string{"api_key"}},
	{Type: "elastic_email", Name: "Elastic Email", NameEN: "Elastic Email", Description: "Elastic Email 营销邮件", Fields: []string{"api_key"}},
	{Type: "aliyun_dm", Name: "阿里云邮件推送", NameEN: "Aliyun DirectMail", Description: "阿里云 DirectMail", Fields: []string{"access_key", "secret_key", "region", "account_name"}},
	{Type: "tencent_ses", Name: "腾讯云邮件", NameEN: "Tencent Cloud SES", Description: "腾讯云邮件服务 SES", Fields: []string{"secret_id", "secret_key", "region"}},
}

type bulkConfig map[string]string

func parseBulkConfig(raw string) bulkConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return bulkConfig{}
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return bulkConfig{}
	}
	return bulkConfig(m)
}

func (c bulkConfig) get(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(c[k]); v != "" {
			return v
		}
	}
	return ""
}

func maskBulkConfig(raw string) map[string]string {
	m := parseBulkConfig(raw)
	out := make(map[string]string, len(m))
	for k, v := range m {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "secret") || strings.Contains(lk, "password") || strings.Contains(lk, "token") || lk == "api_key" || strings.HasSuffix(lk, "_key") {
			if len(v) > 4 {
				out[k] = v[:2] + "****" + v[len(v)-2:]
			} else if v != "" {
				out[k] = "****"
			}
			continue
		}
		out[k] = v
	}
	return out
}

func mergeBulkConfig(existing string, patch map[string]string) bulkConfig {
	base := parseBulkConfig(existing)
	for k, v := range patch {
		v = strings.TrimSpace(v)
		if v == "" || v == "****" || strings.Contains(v, "****") {
			continue
		}
		base[k] = v
	}
	return base
}

func bulkConfigJSON(c bulkConfig) string {
	b, _ := json.Marshal(c)
	return string(b)
}

type outboundMessage struct {
	From, FromName, ReplyTo, To, ToName, Subject, BodyText, BodyHTML string
}

func (m outboundMessage) fromHeader() string {
	from := strings.TrimSpace(m.From)
	name := strings.TrimSpace(m.FromName)
	if name != "" {
		return fmt.Sprintf("%s <%s>", name, from)
	}
	return from
}
