package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"outlook-helper/backend/internal/models"
)

// OutlookService Outlook API服务
type OutlookService struct {
	baseURL     string
	apiPassword string
	httpClient  *http.Client
}

const (
	microsoftConsumersOAuthBaseURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0"
	defaultDeviceCodeScope         = "offline_access https://outlook.office.com/IMAP.AccessAsUser.All"
)

// NewOutlookService 创建Outlook服务
func NewOutlookService(baseURL, apiPassword string) *OutlookService {
	return &OutlookService{
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiPassword: strings.TrimSpace(apiPassword),
		httpClient:  &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OutlookAPIResponse Outlook API响应结构
type OutlookAPIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// MailData 邮件数据结构
type MailData struct {
	ID          string    `json:"id"`
	Subject     string    `json:"subject"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Body        string    `json:"body"`
	BodyPreview string    `json:"bodyPreview"`
	IsRead      bool      `json:"isRead"`
	ReceivedAt  time.Time `json:"receivedDateTime"`
	VerifyCode  string    `json:"verifyCode,omitempty"`
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	RefreshToken string      `json:"refresh_token"`
	Error        string      `json:"error,omitempty"`
	Message      string      `json:"message,omitempty"`
	Details      interface{} `json:"details,omitempty"`
}

type microsoftDeviceTokenResponse struct {
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	ExpiresIn        int    `json:"expires_in"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// StartDeviceAuthorization 创建微软设备授权码
func (s *OutlookService) StartDeviceAuthorization(clientID, scope string) (*models.OAuthDeviceCodeResponse, error) {
	clientID = strings.TrimSpace(clientID)
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = defaultDeviceCodeScope
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", scope)

	req, err := http.NewRequest(
		"POST",
		microsoftConsumersOAuthBaseURL+"/devicecode",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("创建设备授权请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求设备授权码失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取设备授权响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("设备授权请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var response models.OAuthDeviceCodeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析设备授权响应失败: %v, 响应内容: %s", err, string(body))
	}

	if response.DeviceCode == "" || response.UserCode == "" || response.VerificationURI == "" {
		return nil, fmt.Errorf("设备授权响应缺少必要字段")
	}
	if response.Interval <= 0 {
		response.Interval = 5
	}

	return &response, nil
}

// PollDeviceToken 轮询设备授权结果
func (s *OutlookService) PollDeviceToken(clientID, deviceCode string) (*models.OAuthDeviceTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("client_id", strings.TrimSpace(clientID))
	form.Set("device_code", strings.TrimSpace(deviceCode))

	req, err := http.NewRequest(
		"POST",
		microsoftConsumersOAuthBaseURL+"/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("创建设备授权轮询请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("轮询设备授权结果失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取设备授权轮询响应失败: %v", err)
	}

	var tokenResponse microsoftDeviceTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("解析设备授权轮询响应失败: %v, 响应内容: %s", err, string(body))
	}

	if tokenResponse.Error != "" {
		status := "failed"
		if tokenResponse.Error == "authorization_pending" || tokenResponse.Error == "slow_down" {
			status = "pending"
		}

		return &models.OAuthDeviceTokenResponse{
			Status:           status,
			Error:            tokenResponse.Error,
			ErrorDescription: tokenResponse.ErrorDescription,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("设备授权轮询失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	if strings.TrimSpace(tokenResponse.RefreshToken) == "" {
		return nil, fmt.Errorf("设备授权成功但未返回refresh_token，请确认scope包含offline_access")
	}

	return &models.OAuthDeviceTokenResponse{
		Status:               "authorized",
		RefreshToken:         tokenResponse.RefreshToken,
		AccessTokenExpiresIn: tokenResponse.ExpiresIn,
		Scope:                tokenResponse.Scope,
	}, nil
}

// RefreshToken 使用当前RefreshToken换取新的RefreshToken
func (s *OutlookService) RefreshToken(email *models.Email) (string, error) {
	requestData := map[string]string{
		"refresh_token": email.RefreshToken,
		"client_id":     email.ClientID,
	}
	s.addOutlookAPIPassword(requestData)

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("序列化刷新令牌请求失败: %v", err)
	}

	requestURL := fmt.Sprintf("%s/api/refresh-token", s.baseURL)
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建刷新令牌请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("刷新令牌请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取刷新令牌响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("刷新令牌API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var response RefreshTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析刷新令牌响应失败: %v, 响应内容: %s", err, string(body))
	}

	if response.Error != "" {
		return "", fmt.Errorf("刷新令牌API返回错误: %s", response.Error)
	}

	newRefreshToken := strings.TrimSpace(response.RefreshToken)
	if newRefreshToken == "" {
		return "", fmt.Errorf("刷新令牌API未返回refresh_token")
	}

	return newRefreshToken, nil
}

// GetLatestMail 获取最新邮件
func (s *OutlookService) GetLatestMail(email *models.Email, mailbox string, responseType string) (*models.OutlookMail, error) {
	// 构建请求体
	requestData := map[string]string{
		"refresh_token": email.RefreshToken,
		"client_id":     email.ClientID,
		"email":         email.EmailAddress,
		"mailbox":       mailbox,
	}
	s.addOutlookAPIPassword(requestData)
	if responseType != "" {
		requestData["response_type"] = responseType
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	requestURL := fmt.Sprintf("%s/api/mail-new", s.baseURL)

	// 创建POST请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 尝试解析邮件数据，支持两种格式：单个对象或数组
	var mailData map[string]interface{}

	// 首先尝试解析为单个对象
	if err := json.Unmarshal(body, &mailData); err != nil {
		// 如果失败，尝试解析为数组格式
		var mailsArray []map[string]interface{}
		if arrayErr := json.Unmarshal(body, &mailsArray); arrayErr != nil {
			// 两种格式都解析失败，记录完整的响应内容
			return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
		}

		// 如果是数组格式，取第一个元素作为最新邮件
		if len(mailsArray) == 0 {
			return nil, fmt.Errorf("API返回空邮件数组")
		}
		mailData = mailsArray[0]
	}

	// 解析时间字段
	var receivedAt time.Time
	if dateStr := getStringFromMap(mailData, "date"); dateStr != "" {
		if parsedTime, err := time.Parse(time.RFC3339, dateStr); err == nil {
			receivedAt = parsedTime
		}
	}

	// 优先获取HTML格式的邮件内容，如果没有则使用text字段
	mailBody := getStringFromMap(mailData, "html")
	if mailBody == "" {
		mailBody = getStringFromMap(mailData, "text")
	}

	mail := &models.OutlookMail{
		ID:         getStringFromMap(mailData, "id"),
		Subject:    getStringFromMap(mailData, "subject"),
		From:       getStringFromMap(mailData, "send"), // API返回的字段名是"send"
		To:         getStringFromMap(mailData, "to"),
		Body:       mailBody, // 优先使用HTML格式，否则使用text字段
		IsRead:     getBoolFromMap(mailData, "isRead"),
		VerifyCode: getStringFromMap(mailData, "verifyCode"),
		ReceivedAt: receivedAt,
	}

	// 解析时间
	if receivedAtStr := getStringFromMap(mailData, "receivedDateTime"); receivedAtStr != "" {
		if parsedTime, err := time.Parse(time.RFC3339, receivedAtStr); err == nil {
			mail.ReceivedAt = parsedTime
		}
	}

	return mail, nil
}

// GetAllMails 获取全部邮件
func (s *OutlookService) GetAllMails(email *models.Email, mailbox string) ([]models.OutlookMail, error) {
	// 构建请求体
	requestData := map[string]string{
		"refresh_token": email.RefreshToken,
		"client_id":     email.ClientID,
		"email":         email.EmailAddress,
		"mailbox":       mailbox,
	}
	s.addOutlookAPIPassword(requestData)

	// 序列化请求体
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	requestURL := fmt.Sprintf("%s/api/mail-all", s.baseURL)

	// 创建POST请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 直接解析邮件数组（API直接返回邮件对象数组，不包装在通用响应中）
	var mailsData []map[string]interface{}
	if err := json.Unmarshal(body, &mailsData); err != nil {
		// 记录完整的响应内容以便调试
		return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	var mails []models.OutlookMail
	for _, mailData := range mailsData {
		// 解析时间字段
		var receivedAt time.Time
		if dateStr := getStringFromMap(mailData, "date"); dateStr != "" {
			if parsedTime, err := time.Parse(time.RFC3339, dateStr); err == nil {
				receivedAt = parsedTime
			}
		}

		// 优先获取HTML格式的邮件内容，如果没有则使用text字段
		mailBody := getStringFromMap(mailData, "html")
		if mailBody == "" {
			mailBody = getStringFromMap(mailData, "text")
		}

		mail := models.OutlookMail{
			ID:         getStringFromMap(mailData, "id"),
			Subject:    getStringFromMap(mailData, "subject"),
			From:       getStringFromMap(mailData, "send"), // API返回的字段名是"send"
			To:         getStringFromMap(mailData, "to"),
			Body:       mailBody, // 优先使用HTML格式，否则使用text字段
			IsRead:     getBoolFromMap(mailData, "isRead"),
			VerifyCode: getStringFromMap(mailData, "verifyCode"),
			ReceivedAt: receivedAt,
		}

		mails = append(mails, mail)
	}

	return mails, nil
}

// ClearInbox 清空收件箱
func (s *OutlookService) ClearInbox(email *models.Email) error {
	// 构建请求体
	requestData := map[string]string{
		"refresh_token": email.RefreshToken,
		"client_id":     email.ClientID,
		"email":         email.EmailAddress,
	}
	s.addOutlookAPIPassword(requestData)

	// 序列化请求体
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	requestURL := fmt.Sprintf("%s/api/process-inbox", s.baseURL)

	// 创建POST请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应（API直接返回消息对象）
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		// 记录完整的响应内容以便调试
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	// 检查是否有错误消息
	if errorMsg := getStringFromMap(response, "error"); errorMsg != "" {
		return fmt.Errorf("API返回错误: %s", errorMsg)
	}

	return nil
}

// ClearJunk 清空垃圾箱
func (s *OutlookService) ClearJunk(email *models.Email) error {
	// 构建请求体
	requestData := map[string]string{
		"refresh_token": email.RefreshToken,
		"client_id":     email.ClientID,
		"email":         email.EmailAddress,
	}
	s.addOutlookAPIPassword(requestData)

	// 序列化请求体
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	requestURL := fmt.Sprintf("%s/api/process-junk", s.baseURL)

	// 创建POST请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应（API直接返回消息对象）
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		// 记录完整的响应内容以便调试
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	// 检查是否有错误消息
	if errorMsg := getStringFromMap(response, "error"); errorMsg != "" {
		return fmt.Errorf("API返回错误: %s", errorMsg)
	}

	return nil
}

// ValidateEmailCredentials 验证邮箱凭据
func (s *OutlookService) ValidateEmailCredentials(email *models.Email) error {
	// 尝试获取最新邮件来验证凭据
	_, err := s.GetLatestMail(email, "INBOX", "json")
	if err != nil {
		// 为验证失败提供更详细的错误信息
		return fmt.Errorf("验证邮箱 %s 凭据失败: %v", email.EmailAddress, err)
	}
	return nil
}

// 辅助函数
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (s *OutlookService) addOutlookAPIPassword(requestData map[string]string) {
	if s.apiPassword != "" {
		requestData["password"] = s.apiPassword
	}
}
