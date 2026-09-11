package app

import (
	"sort"
	"strings"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/config"
	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

type xiaoyuCapabilitySearchResult struct {
	Query   string                             `json:"query"`
	Modules []xiaoyuhost.SystemModuleKnowledge `json:"modules"`
	Tools   []xiaoyuCapabilityToolMatch        `json:"tools"`
	Note    string                             `json:"note"`
}

type xiaoyuCapabilityToolMatch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Risk        string `json:"risk"`
	Source      string `json:"source,omitempty"`
	Available   bool   `json:"available"`
}

func (a *Application) xiaoyuSystemModuleCatalog() []xiaoyuhost.SystemModuleSummary {
	modules := append([]config.ModuleConfig(nil), a.platformConfig.Modules.Modules...)
	sort.SliceStable(modules, func(i, j int) bool {
		if modules[i].Category == modules[j].Category {
			return modules[i].ID < modules[j].ID
		}
		return modules[i].Category < modules[j].Category
	})
	out := make([]xiaoyuhost.SystemModuleSummary, 0, len(modules))
	for _, module := range modules {
		out = append(out, xiaoyuhost.SystemModuleSummary{
			ID: module.ID, Name: module.Name, Status: module.Status, Route: module.Route,
		})
	}
	return out
}

func (a *Application) xiaoyuSystemModules(goal string, run xiaoyuhost.RunContext) []xiaoyuhost.SystemModuleKnowledge {
	return selectXiaoYuModules(a.platformConfig.Modules.Modules, strings.TrimSpace(goal+" "+run.GameID+" "+run.UIRoute), 12)
}

func selectXiaoYuModules(modules []config.ModuleConfig, query string, limit int) []xiaoyuhost.SystemModuleKnowledge {
	if limit <= 0 {
		limit = 12
	}
	type scored struct {
		module config.ModuleConfig
		score  int
	}
	q := strings.ToLower(strings.TrimSpace(query))
	items := make([]scored, 0, len(modules))
	for _, module := range modules {
		score := moduleSearchScore(q, module)
		if module.ID == "ai" {
			score += 1000
		}
		if q != "" && strings.Contains(q, strings.ToLower(module.Route)) {
			score += 100
		}
		if score > 0 {
			items = append(items, scored{module: module, score: score})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
	out := make([]xiaoyuhost.SystemModuleKnowledge, 0, minIntApp(len(items), limit))
	for _, item := range items {
		if len(out) >= limit {
			break
		}
		out = append(out, xiaoyuhost.SystemModuleKnowledge{
			ID:       item.module.ID,
			Name:     item.module.Name,
			Category: item.module.Category,
			Route:    item.module.Route,
			Status:   item.module.Status,
			Phase:    item.module.Phase,
			Features: append([]string(nil), item.module.Features...),
		})
	}
	return out
}

func moduleSearchScore(query string, module config.ModuleConfig) int {
	if query == "" {
		if module.ID == "ai" {
			return 100
		}
		return 0
	}
	parts := []string{module.ID, module.Name, module.Category, module.Route, module.Status, module.Phase, strings.Join(module.Features, " ")}
	score := 0
	for _, raw := range parts {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if strings.Contains(query, value) {
			score += 20
		}
		for _, token := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ' ' || r == ',' || r == '，' || r == '/' || r == '|' || r == ':' || r == '：' || r == '-' || r == '_' || r == '.'
		}) {
			token = strings.TrimSpace(token)
			if len([]rune(token)) >= 2 && strings.Contains(query, token) {
				score += 3
			}
		}
	}
	return score
}

func (a *Application) searchXiaoYuCapabilities(query string) xiaoyuCapabilitySearchResult {
	query = strings.TrimSpace(query)
	result := xiaoyuCapabilitySearchResult{
		Query: query,
		Note:  "modules 描述 AGMP 产品功能与实现阶段；只有当前 Tool Registry 中 available=true 的 Tool 才能由 XiaoYu 直接执行。skeleton/planned 不能当作已实现能力。",
	}
	result.Modules = selectXiaoYuModules(a.platformConfig.Modules.Modules, query, 16)
	if a.xiaoyuTools == nil {
		return result
	}
	type scoredTool struct {
		spec  xiaoyucontract.ToolSpec
		score int
	}
	q := strings.ToLower(query)
	matches := make([]scoredTool, 0)
	for _, spec := range a.xiaoyuTools.List() {
		if !spec.XiaoYu {
			continue
		}
		surface := strings.ToLower(strings.Join([]string{spec.Name, spec.Description, spec.Category, spec.Source}, " "))
		score := 0
		if q == "" {
			score = 1
		} else {
			for _, token := range strings.FieldsFunc(q, func(r rune) bool {
				return r == ' ' || r == ',' || r == '，' || r == '/' || r == '|' || r == ':' || r == '：'
			}) {
				token = strings.TrimSpace(token)
				if token != "" && strings.Contains(surface, token) {
					score += 4
				}
			}
			if strings.Contains(surface, q) {
				score += 20
			}
		}
		if score > 0 {
			matches = append(matches, scoredTool{spec: spec, score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	for _, item := range matches {
		if len(result.Tools) >= 16 {
			break
		}
		result.Tools = append(result.Tools, xiaoyuCapabilityToolMatch{
			Name: item.spec.Name, Description: item.spec.Description, Category: item.spec.Category,
			Risk: string(item.spec.Risk), Source: item.spec.Source, Available: true,
		})
	}
	return result
}

func minIntApp(a, b int) int {
	if a < b {
		return a
	}
	return b
}
