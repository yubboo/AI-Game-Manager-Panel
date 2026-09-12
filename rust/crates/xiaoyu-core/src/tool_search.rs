use xiaoyu_protocol::{ToolSearchHit, ToolSearchRequest, ToolSearchResponse, ToolSpec};

const DEFAULT_LIMIT: usize = 16;
const MAX_LIMIT: usize = 64;

/// Search the model-visible Tool catalog without exposing execution authority.
///
/// XiaoYu uses this as a capability-discovery primitive. A hit means “this Tool
/// is relevant to the current goal”, not “this Tool is approved to execute”.
/// Approval, RBAC, sandbox and domain validation remain separate concerns.
pub fn search_tools(request: ToolSearchRequest) -> ToolSearchResponse {
    let query = normalize(&request.query);
    let terms = search_terms(&query);
    let limit = request.limit.clamp(1, MAX_LIMIT);
    let limit = if request.limit == 0 {
        DEFAULT_LIMIT
    } else {
        limit
    };

    let mut hits = request
        .tools
        .into_iter()
        .filter(|tool| tool.xiaoyu)
        .filter_map(|tool| {
            let score = score_tool(&tool, &query, &terms);
            (score > 0).then_some(ToolSearchHit { tool, score })
        })
        .collect::<Vec<_>>();

    hits.sort_by(|left, right| {
        right
            .score
            .cmp(&left.score)
            .then_with(|| left.tool.name.cmp(&right.tool.name))
    });
    hits.truncate(limit);

    ToolSearchResponse {
        query: request.query.trim().to_string(),
        hits,
    }
}

fn score_tool(tool: &ToolSpec, query: &str, terms: &[String]) -> u32 {
    if query.is_empty() {
        return 1;
    }

    let name = normalize(&tool.name);
    let description = normalize(&tool.description);
    let category = normalize(&tool.category);
    let source = normalize(&tool.source);
    let surface = format!("{name} {description} {category} {source}");

    let mut score = 0;
    if name == query {
        score += 120;
    } else if name.contains(query) || query.contains(&name) {
        score += 60;
    }
    if category == query {
        score += 45;
    } else if category.contains(query) {
        score += 25;
    }
    if description.contains(query) {
        score += 30;
    }
    if source.contains(query) {
        score += 15;
    }

    for term in terms {
        if name.contains(term) {
            score += 16;
        }
        if category.contains(term) {
            score += 10;
        }
        if description.contains(term) {
            score += 7;
        }
        if source.contains(term) {
            score += 4;
        }
        if surface.as_str() == term.as_str() {
            score += 20;
        }
    }

    score
}

fn normalize(value: &str) -> String {
    value.trim().to_lowercase()
}

fn search_terms(value: &str) -> Vec<String> {
    value
        .split(|ch: char| !ch.is_alphanumeric())
        .map(str::trim)
        .filter(|item| !item.is_empty())
        .map(ToOwned::to_owned)
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use xiaoyu_protocol::RiskLevel;

    fn tool(name: &str, description: &str, category: &str, xiaoyu: bool) -> ToolSpec {
        ToolSpec {
            name: name.to_string(),
            description: description.to_string(),
            risk: RiskLevel::Read,
            category: category.to_string(),
            manual: true,
            xiaoyu,
            parameters: json!({"type":"object"}),
            source: "agmp.test".to_string(),
        }
    }

    #[test]
    fn prefers_name_and_category_matches() {
        let response = search_tools(ToolSearchRequest {
            query: "java runtime".to_string(),
            tools: vec![
                tool(
                    "environment.catalog",
                    "读取 Java Runtime",
                    "environment",
                    true,
                ),
                tool("logs.read", "读取日志", "logs", true),
                tool(
                    "environment.remove_runtime",
                    "移除 Java Runtime",
                    "environment",
                    true,
                ),
            ],
            limit: 8,
        });
        assert_eq!(response.hits.len(), 2);
        assert!(response.hits[0].tool.name.starts_with("environment."));
    }

    #[test]
    fn hides_tools_not_exposed_to_xiaoyu() {
        let response = search_tools(ToolSearchRequest {
            query: "runtime".to_string(),
            tools: vec![
                tool("process.run", "兼容进程入口", "runtime", false),
                tool("shell.exec", "受控 Shell", "runtime", true),
            ],
            limit: 8,
        });
        assert_eq!(response.hits.len(), 1);
        assert_eq!(response.hits[0].tool.name, "shell.exec");
    }

    #[test]
    fn empty_query_returns_bounded_visible_catalog() {
        let response = search_tools(ToolSearchRequest {
            query: String::new(),
            tools: (0..30)
                .map(|index| tool(&format!("tool.{index}"), "demo", "test", true))
                .collect(),
            limit: 0,
        });
        assert_eq!(response.hits.len(), DEFAULT_LIMIT);
    }
}
