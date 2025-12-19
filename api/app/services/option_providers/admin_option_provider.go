package option_providers

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
	"github.com/spf13/cast"

	"goravel/app/models"
)

type AdminOptionProvider struct{}

func NewAdminOptionProvider() *AdminOptionProvider {
	return &AdminOptionProvider{}
}

func (p *AdminOptionProvider) GetOptions(ctx http.Context) (map[string]any, error) {
	var admins []models.Admin

	// 检查是否只返回客服人员
	customerServiceOnly := ctx.Request().Query("customer_service_only", "") == "1" || ctx.Request().Query("customer_service_only", "") == "true"

	// 排除开发者ID
	developerIDsStr := facades.Config().GetString("admin.developer_ids", "2")
	developerIDs := parseProtectedIDs(developerIDsStr)

	query := facades.Orm().Query().Where("status", 1)

	// 如果只需要客服人员，先获取客服角色的管理员ID
	if customerServiceOnly {
		// 先获取客服角色的ID
		var customerServiceRole models.Role
		if err := facades.Orm().Query().Where("slug", "customer-service").Where("status", 1).First(&customerServiceRole); err != nil {
			// 如果没有客服角色，返回空列表
			return map[string]any{
				"options": []map[string]any{},
			}, nil
		}

		// 获取所有启用的、具有客服角色的管理员ID
		var adminIDs []uint
		if err := facades.Orm().Query().
			Table("admin_role").
			Select("admin_id").
			Where("role_id", customerServiceRole.ID).
			Pluck("admin_id", &adminIDs); err != nil {
			return nil, err
		}

		if len(adminIDs) == 0 {
			return map[string]any{
				"options": []map[string]any{},
			}, nil
		}

		query = query.Where("id IN ?", adminIDs)
	}

	if len(developerIDs) > 0 {
		query = query.Where("id NOT IN ?", developerIDs)
	}

	if err := query.Order("id asc").Get(&admins); err != nil {
		return nil, err
	}

	var options []map[string]any
	for _, admin := range admins {
		label := admin.Username
		if admin.Nickname != "" {
			label = admin.Nickname + " (" + admin.Username + ")"
		}
		options = append(options, map[string]any{
			"label": label,
			"value": cast.ToString(admin.ID),
		})
	}

	return map[string]any{
		"options": options,
	}, nil
}

// parseProtectedIDs 解析受保护的管理员ID字符串（支持逗号分隔）
func parseProtectedIDs(idsStr string) []uint {
	var ids []uint
	if idsStr == "" {
		return ids
	}

	// 使用字符串分割
	parts := str.Of(idsStr).Split(",")
	for _, part := range parts {
		part = str.Of(part).Trim().String()
		if !str.Of(part).IsEmpty() {
			if id := cast.ToUint(part); id > 0 {
				ids = append(ids, id)
			}
		}
	}

	return ids
}
