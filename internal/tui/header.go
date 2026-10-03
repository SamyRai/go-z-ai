package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// renderHeader builds the top line: the app badge on the left and a
// right-aligned cluster of context badges (account, type, plan, model). On
// narrow terminals the less essential badges drop first.
func (m *rootModel) renderHeader(width int) string {
	left := uistyle.HeaderApp.Render("go-z-ai")

	var accountBadge, typeBadge, planBadge, modelBadge string
	if acct, ok := m.session.activeAccount(); ok {
		accountBadge = uistyle.RenderBadge("account", acct.Name, uistyle.BadgeValue)
		typeBadge = uistyle.RenderBadge("type", accountTypeLabel(acct.Type), uistyle.BadgeValue)
	} else {
		accountBadge = uistyle.RenderBadge("account", "none", uistyle.BadgeWarn)
	}
	if m.session != nil && coding.IsValidPlan(m.session.plan) {
		planBadge = uistyle.RenderBadge("plan", coding.RegionLabel(m.session.plan), uistyle.BadgeOK)
	}
	if g, ok := m.screens[tabChat].(chatModelGetter); ok && g.ModelID() != "" {
		modelBadge = uistyle.RenderBadge("model", g.ModelID(), uistyle.BadgeValue)
	}

	badges := joinBadges("  ", accountBadge, typeBadge, planBadge, modelBadge)
	switch {
	case width > 0 && width < fullTabBarWidth()-30:
		badges = accountBadge
	case width > 0 && width < fullTabBarWidth():
		badges = joinBadges("  ", accountBadge, modelBadge)
	}
	if badges == "" {
		return left
	}
	spacer := strings.Repeat(" ", max(width-lipgloss.Width(left)-lipgloss.Width(badges)-1, 1))
	return left + spacer + badges
}

// accountTypeLabel renders an account type compactly for a badge.
func accountTypeLabel(t client.AccountType) string {
	switch t {
	case client.AccountTypePayAsYouGo:
		return "pay-as-you-go"
	case client.AccountTypeCodingPlan:
		return "coding-plan"
	}
	return string(t)
}

// joinBadges joins the non-empty badges with sep.
func joinBadges(sep string, badges ...string) string {
	var kept []string
	for _, b := range badges {
		if b != "" {
			kept = append(kept, b)
		}
	}
	return strings.Join(kept, sep)
}
