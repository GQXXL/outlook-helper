package database

import (
	"database/sql"
	"strings"

	"outlook-helper/backend/internal/models"
)

// AccessCodeRepository 授权码数据库操作
type AccessCodeRepository struct {
	db *sql.DB
}

// NewAccessCodeRepository 创建授权码仓库
func NewAccessCodeRepository(db *sql.DB) *AccessCodeRepository {
	return &AccessCodeRepository{db: db}
}

// CreateAccessCode 创建授权码
func (r *AccessCodeRepository) CreateAccessCode(accessCode *models.AccessCode, emailIDs []int) (*models.AccessCode, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO access_codes (name, code_hash, role, enabled, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, accessCode.Name, accessCode.CodeHash, accessCode.Role, accessCode.Enabled, accessCode.ExpiresAt)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err := replaceAccessCodeEmails(tx, int(id), emailIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetAccessCodeByID(int(id))
}

// GetAccessCodeByID 根据ID获取授权码
func (r *AccessCodeRepository) GetAccessCodeByID(id int) (*models.AccessCode, error) {
	accessCode := &models.AccessCode{}
	var expiresAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, name, code_hash, role, enabled, expires_at, created_at, updated_at
		FROM access_codes
		WHERE id = ?
	`, id).Scan(
		&accessCode.ID,
		&accessCode.Name,
		&accessCode.CodeHash,
		&accessCode.Role,
		&accessCode.Enabled,
		&expiresAt,
		&accessCode.CreatedAt,
		&accessCode.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		accessCode.ExpiresAt = &expiresAt.Time
	}

	emailIDs, err := r.GetEmailIDs(accessCode.ID)
	if err == nil {
		accessCode.EmailIDs = emailIDs
		accessCode.EmailCount = len(emailIDs)
	}

	return accessCode, nil
}

// GetEnabledAccessCodeByHash 根据授权码hash获取启用的授权码
func (r *AccessCodeRepository) GetEnabledAccessCodeByHash(codeHash string) (*models.AccessCode, error) {
	accessCode := &models.AccessCode{}
	var expiresAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, name, code_hash, role, enabled, expires_at, created_at, updated_at
		FROM access_codes
		WHERE code_hash = ?
		  AND enabled = 1
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`, codeHash).Scan(
		&accessCode.ID,
		&accessCode.Name,
		&accessCode.CodeHash,
		&accessCode.Role,
		&accessCode.Enabled,
		&expiresAt,
		&accessCode.CreatedAt,
		&accessCode.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		accessCode.ExpiresAt = &expiresAt.Time
	}

	emailIDs, err := r.GetEmailIDs(accessCode.ID)
	if err == nil {
		accessCode.EmailIDs = emailIDs
		accessCode.EmailCount = len(emailIDs)
	}

	return accessCode, nil
}

// ListAccessCodes 获取授权码列表
func (r *AccessCodeRepository) ListAccessCodes() ([]models.AccessCode, error) {
	rows, err := r.db.Query(`
		SELECT ac.id, ac.name, ac.code_hash, ac.role, ac.enabled, ac.expires_at,
		       ac.created_at, ac.updated_at, COUNT(ace.email_id) AS email_count
		FROM access_codes ac
		LEFT JOIN access_code_emails ace ON ace.access_code_id = ac.id
		GROUP BY ac.id
		ORDER BY ac.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accessCodes []models.AccessCode
	for rows.Next() {
		var accessCode models.AccessCode
		var expiresAt sql.NullTime
		if err := rows.Scan(
			&accessCode.ID,
			&accessCode.Name,
			&accessCode.CodeHash,
			&accessCode.Role,
			&accessCode.Enabled,
			&expiresAt,
			&accessCode.CreatedAt,
			&accessCode.UpdatedAt,
			&accessCode.EmailCount,
		); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			accessCode.ExpiresAt = &expiresAt.Time
		}

		emailIDs, err := r.GetEmailIDs(accessCode.ID)
		if err == nil {
			accessCode.EmailIDs = emailIDs
		}

		accessCodes = append(accessCodes, accessCode)
	}

	return accessCodes, rows.Err()
}

// UpdateAccessCode 更新授权码基础信息和邮箱绑定
func (r *AccessCodeRepository) UpdateAccessCode(id int, req *models.UpdateAccessCodeRequest) (*models.AccessCode, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE access_codes
		SET name = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, req.Name, req.Enabled, id)
	if err != nil {
		return nil, err
	}

	if err := replaceAccessCodeEmails(tx, id, req.EmailIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetAccessCodeByID(id)
}

// RotateCode 更新授权码hash
func (r *AccessCodeRepository) RotateCode(id int, codeHash string) (*models.AccessCode, error) {
	_, err := r.db.Exec(`
		UPDATE access_codes
		SET code_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, codeHash, id)
	if err != nil {
		return nil, err
	}
	return r.GetAccessCodeByID(id)
}

// DeleteAccessCode 删除授权码
func (r *AccessCodeRepository) DeleteAccessCode(id int) error {
	_, err := r.db.Exec(`DELETE FROM access_codes WHERE id = ?`, id)
	return err
}

// GetEmailIDs 获取授权码绑定的邮箱ID
func (r *AccessCodeRepository) GetEmailIDs(accessCodeID int) ([]int, error) {
	rows, err := r.db.Query(`
		SELECT email_id
		FROM access_code_emails
		WHERE access_code_id = ?
		ORDER BY email_id
	`, accessCodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emailIDs []int
	for rows.Next() {
		var emailID int
		if err := rows.Scan(&emailID); err != nil {
			return nil, err
		}
		emailIDs = append(emailIDs, emailID)
	}

	return emailIDs, rows.Err()
}

// IsEmailAllowed 检查授权码是否允许访问邮箱
func (r *AccessCodeRepository) IsEmailAllowed(accessCodeID, emailID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM access_code_emails
		WHERE access_code_id = ? AND email_id = ?
	`, accessCodeID, emailID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func replaceAccessCodeEmails(tx *sql.Tx, accessCodeID int, emailIDs []int) error {
	if _, err := tx.Exec(`DELETE FROM access_code_emails WHERE access_code_id = ?`, accessCodeID); err != nil {
		return err
	}

	if len(emailIDs) == 0 {
		return nil
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO access_code_emails (access_code_id, email_id, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, emailID := range emailIDs {
		if _, err := stmt.Exec(accessCodeID, emailID); err != nil {
			if strings.Contains(err.Error(), "FOREIGN KEY") {
				continue
			}
			return err
		}
	}

	return nil
}
