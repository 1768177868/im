package seeders

import (
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type AdminSeeder struct {
}

func (s *AdminSeeder) Signature() string {
	return "AdminSeeder"
}

func (s *AdminSeeder) Run() error {
	// 创建超级管理员
	hashedPassword, _ := facades.Hash().Make("admin123")
	var superAdmin models.Admin
	if err := facades.Orm().Query().Where("username", "admin").First(&superAdmin); err != nil {
		// 不存在则创建
		superAdmin = models.Admin{
			Username: "admin",
			Password: hashedPassword,
			Nickname: "超级管理员",
			Status:   1,
		}
		facades.Orm().Query().Create(&superAdmin)
		// 重新查询确保获取完整的记录
		facades.Orm().Query().Where("username", "admin").First(&superAdmin)
	}

	// 创建开发者管理员（受保护，不显示在列表中）
	developerPassword, _ := facades.Hash().Make("developer123")
	var developerAdmin models.Admin
	if err := facades.Orm().Query().Where("username", "developer").First(&developerAdmin); err != nil {
		// 不存在则创建
		developerAdmin = models.Admin{
			Username: "developer",
			Password: developerPassword,
			Nickname: "开发者管理员",
			Status:   1,
		}
		facades.Orm().Query().Create(&developerAdmin)
		// 重新查询确保获取完整的记录
		facades.Orm().Query().Where("username", "developer").First(&developerAdmin)
	}

	// 创建部门
	var rootDept models.Department
	if err := facades.Orm().Query().Where("name", "总公司").First(&rootDept); err != nil {
		// 不存在则创建
		rootDept = models.Department{
			Name:   "总公司",
			Code:   "ROOT",
			Status: 1,
			Sort:   0,
		}
		if err := facades.Orm().Query().Create(&rootDept); err != nil {
			facades.Log().Errorf("创建总公司部门失败: %v", err)
			// 如果创建失败，尝试重新查询（可能已存在）
			if err := facades.Orm().Query().Where("name", "总公司").First(&rootDept); err != nil {
				facades.Log().Errorf("查询总公司部门失败: %v", err)
				return err
			}
		} else {
			// 重新查询确保获取完整的记录（包括ID）
			if err := facades.Orm().Query().Where("name", "总公司").First(&rootDept); err != nil {
				facades.Log().Errorf("重新查询总公司部门失败: %v", err)
				return err
			}
		}
	} else {
		// 存在则更新（如果字段为空）
		update := false
		if rootDept.Name == "" {
			rootDept.Name = "总公司"
			update = true
		}
		if rootDept.Code == "" {
			rootDept.Code = "ROOT"
			update = true
		}
		if rootDept.Status == 0 {
			rootDept.Status = 1
			update = true
		}
		if update && rootDept.ID > 0 {
			if _, err := facades.Orm().Query().Model(&models.Department{}).Where("id", rootDept.ID).Update(map[string]interface{}{
				"name":   rootDept.Name,
				"code":   rootDept.Code,
				"status": rootDept.Status,
			}); err != nil {
				facades.Log().Errorf("更新总公司部门失败: %v", err)
			}
		}
	}

	var itDept models.Department
	if err := facades.Orm().Query().Where("name", "技术部").First(&itDept); err != nil {
		// 不存在则创建
		itDept = models.Department{
			ParentID: rootDept.ID,
			Name:     "技术部",
			Code:     "IT",
			Status:   1,
			Sort:     1,
		}
		facades.Orm().Query().Create(&itDept)
		// 重新查询确保获取完整的记录
		facades.Orm().Query().Where("name", "技术部").First(&itDept)
	} else {
		// 存在则更新（如果字段为空）
		update := false
		if itDept.Name == "" {
			itDept.Name = "技术部"
			update = true
		}
		if itDept.Code == "" {
			itDept.Code = "IT"
			update = true
		}
		if itDept.ParentID == 0 {
			itDept.ParentID = rootDept.ID
			update = true
		}
		if itDept.Status == 0 {
			itDept.Status = 1
			update = true
		}
		if update {
			facades.Orm().Query().Model(&models.Department{}).Where("id", itDept.ID).Update(map[string]interface{}{
				"name":      itDept.Name,
				"code":      itDept.Code,
				"parent_id": itDept.ParentID,
				"status":    itDept.Status,
			})
		}
	}

	// 创建角色
	var superRole models.Role
	// FirstOrCreate: 第一个参数是查询条件（只包含唯一字段），第二个参数是创建时的默认值
	if err := facades.Orm().Query().Where("name", "超级管理员").First(&superRole); err != nil {
		// 不存在则创建
		superRole = models.Role{
			Name:        "超级管理员",
			Slug:        "super-admin",
			Description: "拥有所有权限",
			Status:      1,
			Sort:        0,
		}
		facades.Orm().Query().Create(&superRole)
	} else {
		// 存在则更新其他字段（如果为空或需要更新）
		update := false
		if superRole.Slug == "" {
			superRole.Slug = "super-admin"
			update = true
		}
		if superRole.Description == "" {
			superRole.Description = "拥有所有权限"
			update = true
		}
		if superRole.Status == 0 {
			superRole.Status = 1
			update = true
		}
		if superRole.Sort == 0 {
			superRole.Sort = 0
			update = true
		}
		if update {
			facades.Orm().Query().Save(&superRole)
		}
	}

	var adminRole models.Role
	if err := facades.Orm().Query().Where("name", "管理员").First(&adminRole); err != nil {
		// 不存在则创建
		adminRole = models.Role{
			Name:        "管理员",
			Slug:        "admin",
			Description: "普通管理员",
			Status:      1,
			Sort:        1,
		}
		facades.Orm().Query().Create(&adminRole)
	} else {
		// 存在则更新其他字段（如果为空或需要更新）
		update := false
		if adminRole.Slug == "" {
			adminRole.Slug = "admin"
			update = true
		}
		if adminRole.Description == "" {
			adminRole.Description = "普通管理员"
			update = true
		}
		if adminRole.Status == 0 {
			adminRole.Status = 1
			update = true
		}
		if adminRole.Sort == 0 {
			adminRole.Sort = 1
			update = true
		}
		if update {
			facades.Orm().Query().Save(&adminRole)
		}
	}

	// 关联超级管理员和超级角色
	if superAdmin.ID > 0 {
		facades.Orm().Query().Model(&superAdmin).Association("Roles").Replace([]models.Role{superRole})
	}

	// 给开发者管理员分配 super-admin 角色
	if developerAdmin.ID > 0 {
		facades.Orm().Query().Model(&developerAdmin).Association("Roles").Replace([]models.Role{superRole})
	}

	// super-admin 角色不需要分配权限和菜单，因为它在权限中间件中会跳过权限检查
	// 在获取用户信息时，会特殊处理 super-admin 角色，返回所有菜单用于前端显示

	// 创建演示账户角色（只允许查看，不允许编辑创建删除）
	var demoRole models.Role
	if err := facades.Orm().Query().Where("name", "演示账户").First(&demoRole); err != nil {
		// 不存在则创建
		demoRole = models.Role{
			Name:        "演示账户",
			Slug:        "demo",
			Description: "演示账户，只允许查看，不允许编辑、创建、删除",
			Status:      1,
			Sort:        2,
		}
		facades.Orm().Query().Create(&demoRole)
	} else {
		// 存在则更新其他字段（如果为空或需要更新）
		update := false
		if demoRole.Slug == "" {
			demoRole.Slug = "demo"
			update = true
		}
		if demoRole.Description == "" {
			demoRole.Description = "演示账户，只允许查看，不允许编辑、创建、删除"
			update = true
		}
		if demoRole.Status == 0 {
			demoRole.Status = 1
			update = true
		}
		if demoRole.Sort == 0 {
			demoRole.Sort = 2
			update = true
		}
		if update {
			facades.Orm().Query().Save(&demoRole)
		}
	}

	// 给演示角色分配所有查看权限（index 和 show）
	var viewPermissions []models.Permission
	if err := facades.Orm().Query().Where("slug LIKE ?", "%.index").OrWhere("slug LIKE ?", "%.show").Find(&viewPermissions); err == nil {
		facades.Orm().Query().Model(&demoRole).Association("Permissions").Replace(viewPermissions)
	}

	// 给演示角色分配所有菜单（用于前端显示）
	var allMenus []models.Menu
	if err := facades.Orm().Query().Where("status", 1).Find(&allMenus); err == nil {
		facades.Orm().Query().Model(&demoRole).Association("Menus").Replace(allMenus)
	}

	// 创建演示账户
	demoPassword, _ := facades.Hash().Make("demo123")
	var demoAdmin models.Admin
	if err := facades.Orm().Query().Where("username", "demo").First(&demoAdmin); err != nil {
		// 不存在则创建
		demoAdmin = models.Admin{
			Username: "demo",
			Password: demoPassword,
			Nickname: "演示账户",
			Status:   1,
		}
		facades.Orm().Query().Create(&demoAdmin)
		// 重新查询确保获取完整的记录
		facades.Orm().Query().Where("username", "demo").First(&demoAdmin)
	}

	// 给演示账户分配演示角色
	if demoAdmin.ID > 0 {
		facades.Orm().Query().Model(&demoAdmin).Association("Roles").Replace([]models.Role{demoRole})
	}

	// 创建客服部门
	// 确保 rootDept 已创建且ID有效
	if rootDept.ID == 0 {
		facades.Log().Errorf("总公司部门ID无效，无法创建客服部门")
		return nil
	}

	var customerServiceDept models.Department
	facades.Orm().Query().Where("name", "客服部").First(&customerServiceDept)

	// 使用 ID 判断是否存在，而不是依赖 First 的错误返回
	if customerServiceDept.ID == 0 {
		// 不存在则创建
		customerServiceDept = models.Department{
			ParentID: rootDept.ID,
			Name:     "客服部",
			Code:     "CS",
			Status:   1,
			Sort:     2,
		}
		if err := facades.Orm().Query().Create(&customerServiceDept); err != nil {
			facades.Log().Errorf("创建客服部门失败: %v", err)
		}
		// 创建后重新查询确保获取完整的记录（包括ID）
		facades.Orm().Query().Where("name", "客服部").First(&customerServiceDept)
		if customerServiceDept.ID == 0 {
			facades.Log().Errorf("创建客服部门后仍无法获取ID")
			return nil
		}
		facades.Log().Infof("客服部门创建成功，ID: %d", customerServiceDept.ID)
	} else {
		// 存在则更新（如果字段为空）
		update := false
		if customerServiceDept.Code == "" {
			customerServiceDept.Code = "CS"
			update = true
		}
		if customerServiceDept.ParentID == 0 {
			customerServiceDept.ParentID = rootDept.ID
			update = true
		}
		if customerServiceDept.Status == 0 {
			customerServiceDept.Status = 1
			update = true
		}
		if update {
			facades.Orm().Query().Model(&models.Department{}).Where("id", customerServiceDept.ID).Update(map[string]interface{}{
				"code":      customerServiceDept.Code,
				"parent_id": customerServiceDept.ParentID,
				"status":    customerServiceDept.Status,
			})
		}
		facades.Log().Infof("客服部门已存在，ID: %d", customerServiceDept.ID)
	}

	// 创建客服角色
	var customerServiceRole models.Role
	facades.Orm().Query().Where("slug", "customer-service").First(&customerServiceRole)

	// 使用 ID 判断是否存在
	if customerServiceRole.ID == 0 {
		// 不存在则创建
		customerServiceRole = models.Role{
			Name:        "客服",
			Slug:        "customer-service",
			Description: "客服人员，负责处理客户咨询",
			Status:      1,
			Sort:        3,
		}
		if err := facades.Orm().Query().Create(&customerServiceRole); err != nil {
			facades.Log().Errorf("创建客服角色失败: %v", err)
		}
		// 创建后重新查询确保获取完整的记录
		facades.Orm().Query().Where("slug", "customer-service").First(&customerServiceRole)
		if customerServiceRole.ID == 0 {
			facades.Log().Errorf("创建客服角色后仍无法获取ID")
			return nil
		}
		facades.Log().Infof("客服角色创建成功，ID: %d", customerServiceRole.ID)
	} else {
		// 存在则更新其他字段（如果为空或需要更新）
		update := false
		if customerServiceRole.Name == "" {
			customerServiceRole.Name = "客服"
			update = true
		}
		if customerServiceRole.Description == "" {
			customerServiceRole.Description = "客服人员，负责处理客户咨询"
			update = true
		}
		if customerServiceRole.Status == 0 {
			customerServiceRole.Status = 1
			update = true
		}
		if update {
			facades.Orm().Query().Model(&models.Role{}).Where("id", customerServiceRole.ID).Update(map[string]interface{}{
				"name":        customerServiceRole.Name,
				"description": customerServiceRole.Description,
				"status":      customerServiceRole.Status,
			})
		}
		facades.Log().Infof("客服角色已存在，ID: %d", customerServiceRole.ID)
	}

	// 给客服角色分配客服管理相关权限
	var customerPermissions []models.Permission
	if err := facades.Orm().Query().Where("slug LIKE ?", "customer.%").Find(&customerPermissions); err == nil {
		facades.Orm().Query().Model(&customerServiceRole).Association("Permissions").Replace(customerPermissions)
	}

	// 给客服角色分配客服管理菜单
	var customerMenus []models.Menu
	if err := facades.Orm().Query().Where("slug LIKE ?", "customer%").Find(&customerMenus); err == nil {
		facades.Orm().Query().Model(&customerServiceRole).Association("Menus").Replace(customerMenus)
	}

	// 创建默认客服账号
	// 默认账号信息：
	// - 用户名：customer-service
	// - 密码：cs123456
	// - 昵称：客服小助手
	// - 部门：客服部
	// - 角色：客服（customer-service）

	// 确保客服部门和角色ID有效
	if customerServiceDept.ID == 0 {
		facades.Log().Errorf("客服部门ID无效，无法创建客服账号")
		return nil
	}
	if customerServiceRole.ID == 0 {
		facades.Log().Errorf("客服角色ID无效，无法创建客服账号")
		return nil
	}

	facades.Log().Infof("准备创建客服账号，部门ID: %d, 角色ID: %d", customerServiceDept.ID, customerServiceRole.ID)

	customerServicePassword, _ := facades.Hash().Make("cs123456")
	var customerServiceAdmin models.Admin
	facades.Orm().Query().Where("username", "customer-service").First(&customerServiceAdmin)

	// 使用 ID 判断是否存在
	if customerServiceAdmin.ID == 0 {
		// 不存在则创建
		customerServiceAdmin = models.Admin{
			Username:     "customer-service",
			Password:     customerServicePassword,
			Nickname:     "客服小助手",
			Status:       1,
			DepartmentID: customerServiceDept.ID,
		}
		if err := facades.Orm().Query().Create(&customerServiceAdmin); err != nil {
			facades.Log().Errorf("创建客服账号失败: %v", err)
		}
		// 创建后重新查询确保获取完整的记录
		facades.Orm().Query().Where("username", "customer-service").First(&customerServiceAdmin)
		if customerServiceAdmin.ID == 0 {
			facades.Log().Errorf("创建客服账号后仍无法获取ID")
			return nil
		}

		// 给客服账号分配客服角色
		facades.Orm().Query().Model(&customerServiceAdmin).Association("Roles").Replace([]models.Role{customerServiceRole})
		facades.Log().Infof("客服账号创建成功，ID: %d, 用户名: customer-service", customerServiceAdmin.ID)
	} else {
		// 存在则更新
		update := false
		if customerServiceAdmin.DepartmentID == 0 {
			customerServiceAdmin.DepartmentID = customerServiceDept.ID
			update = true
		}
		if customerServiceAdmin.Status == 0 {
			customerServiceAdmin.Status = 1
			update = true
		}
		if update {
			facades.Orm().Query().Model(&models.Admin{}).Where("id", customerServiceAdmin.ID).Update(map[string]interface{}{
				"department_id": customerServiceAdmin.DepartmentID,
				"status":        customerServiceAdmin.Status,
			})
		}
		// 确保客服账号有客服角色
		var existingRoles []models.Role
		facades.Orm().Query().Model(&customerServiceAdmin).Association("Roles").Find(&existingRoles)
		hasCustomerServiceRole := false
		for _, role := range existingRoles {
			if role.Slug == "customer-service" {
				hasCustomerServiceRole = true
				break
			}
		}
		if !hasCustomerServiceRole {
			facades.Orm().Query().Model(&customerServiceAdmin).Association("Roles").Append(&customerServiceRole)
		}
		facades.Log().Infof("客服账号已存在，ID: %d", customerServiceAdmin.ID)
	}

	return nil
}
