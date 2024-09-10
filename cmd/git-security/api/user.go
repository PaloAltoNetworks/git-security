package api

import (
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gofiber/fiber/v2"
)

func (a *api) GetUsers(c *fiber.Ctx) error {
	type UserRoles struct {
		Name  string   `json:"name"`
		Roles []string `json:"roles"`
	}
	dedup := treemap.NewWithStringComparator()
	entries, err := a.enforcer.GetGroupingPolicy()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		username := entry[0]
		role := entry[1]
		if userRoles, ok := dedup.Get(username); ok {
			if userRoles, ok := userRoles.(*UserRoles); ok {
				userRoles.Roles = append(userRoles.Roles, role)
			}
		} else {
			dedup.Put(username, &UserRoles{
				Name:  username,
				Roles: []string{role},
			})
		}
	}
	return c.JSON(dedup.Values())
}

func (a *api) GetRoles(c *fiber.Ctx) error {
	roles := make([]string, 0)
	for r := range rolesDefined {
		roles = append(roles, r)
	}
	return c.JSON(roles)
}

func (a *api) UpdateUserRoles(c *fiber.Ctx) error {
	username := c.Params("name")
	if username == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	b := struct {
		Roles []string `json:"roles"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	// validation on the role existence and roles.length > 0
	if len(b.Roles) == 0 {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	for _, role := range b.Roles {
		if _, ok := rolesDefined[role]; !ok {
			return c.SendStatus(fiber.StatusBadRequest)
		}
	}

	// check if the user exists, assuming every user has at least one role
	currentRoles, err := a.enforcer.GetRolesForUser(username)
	if err != nil {
		return err
	}
	if len(currentRoles) == 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	updatedRoles := make(map[string]struct{})
	for _, role := range b.Roles {
		updatedRoles[role] = struct{}{}
	}

	for _, role := range currentRoles {
		if _, ok := updatedRoles[role]; !ok {
			if _, err := a.enforcer.DeleteRoleForUser(username, role); err != nil {
				return err
			}
		} else {
			delete(updatedRoles, role)
		}
	}
	for role := range updatedRoles {
		if _, err := a.enforcer.AddRoleForUser(username, role); err != nil {
			return err
		}
	}
	a.enforcer.SavePolicy()

	return c.SendStatus(200)
}
