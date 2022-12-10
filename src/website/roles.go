package website

import (
	"net/http"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
)

type Role struct {
	Slug        string
	Name        string
	Description string
	Template    string
	Url         string // weird and redundant

	RedirectSlug string
}

var roles = []Role{
	{
		Slug:        "education",
		Name:        "Education Lead",
		Description: "Lead our flagship education initiative and make sure we're putting out a steady stream of content.",
		Template:    "role_education.html",
		Url:         hmnurl.BuildStaffRole("education"),
	},
	{
		Slug:        "advocacy",
		Name:        "Advocacy Lead",
		Description: "Put the Handmade ethos into the world by advocating for better software and better programming practices.",
		Template:    "role_advocacy.html",
		Url:         hmnurl.BuildStaffRole("advocacy"),
	},
	{
		Slug:        "design",
		Name:        "Design Lead",
		Description: "Set the visual direction for everything we do. Make key art, design website features, and more.",
		Template:    "role_design.html",
		Url:         hmnurl.BuildStaffRole("design"),
	},
}

const StaffRolesIndexName = "Roles"

func StaffRolesIndex(c *RequestContext) ResponseData {
	type TemplateData struct {
		templates.BaseData
		Roles []Role
	}

	var res ResponseData
	res.MustWriteTemplate("roles.html", TemplateData{
		BaseData: getBaseDataAutocrumb(c, StaffRolesIndexName),
		Roles:    roles,
	}, c.Perf)
	return res
}

func StaffRole(c *RequestContext) ResponseData {
	type TemplateData struct {
		templates.BaseData
		Role       Role
		OtherRoles []Role
	}

	slug := c.PathParams["slug"]
	role, ok := getRole(slug)
	if !ok {
		return FourOhFour(c) // TODO: Volunteering-specific 404
	}

	if role.RedirectSlug != "" {
		return c.Redirect(hmnurl.BuildStaffRole(role.RedirectSlug), http.StatusSeeOther)
	}

	var otherRoles []Role
	for _, otherRole := range roles {
		if otherRole.Slug == role.Slug {
			continue
		}
		otherRoles = append(otherRoles, otherRole)
		if len(otherRoles) >= 3 {
			break
		}
	}

	var res ResponseData
	res.MustWriteTemplate(role.Template, TemplateData{
		BaseData: getBaseData(c, role.Name, []templates.Breadcrumb{
			{Name: StaffRolesIndexName, Url: hmnurl.BuildStaffRolesIndex()},
			{Name: role.Name, Url: ""},
		}),
		Role:       role,
		OtherRoles: otherRoles,
	}, c.Perf)
	return res
}

func getRole(slug string) (Role, bool) {
	for _, role := range roles {
		if role.Slug == slug {
			return role, true
		}
	}
	return Role{}, false
}
