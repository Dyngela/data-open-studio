use std::collections::{HashMap, HashSet, VecDeque};

use thiserror::Error;

#[derive(Debug, Error)]
pub enum GraphError {
    #[error("cycle detected in pipeline graph")]
    Cycle,
    #[error("node not found: {0}")]
    NodeNotFound(String),
}

/// One level of the execution plan — nodes in the same step can run in parallel.
pub type Step = Vec<String>; // node IDs

/// Edge: data flows from `from` to `to`.
#[derive(Debug, Clone)]
pub struct Edge {
    pub from: String,
    pub to:   String,
}

/// Topological sort (Kahn's algorithm). Returns steps — nodes within a step
/// have no inter-dependencies and can execute concurrently.
pub fn topo_sort(node_ids: &[String], edges: &[Edge]) -> Result<Vec<Step>, GraphError> {
    let mut in_degree: HashMap<&str, usize> = node_ids.iter().map(|id| (id.as_str(), 0)).collect();
    let mut adjacency: HashMap<&str, Vec<&str>> = node_ids.iter().map(|id| (id.as_str(), vec![])).collect();

    for edge in edges {
        *in_degree.entry(edge.to.as_str()).or_insert(0) += 1;
        adjacency.entry(edge.from.as_str()).or_default().push(edge.to.as_str());
    }

    let mut queue: VecDeque<&str> = in_degree.iter()
        .filter(|(_, &d)| d == 0)
        .map(|(id, _)| *id)
        .collect();

    let mut steps: Vec<Step> = Vec::new();
    let mut visited = 0usize;

    while !queue.is_empty() {
        let step: Vec<String> = queue.drain(..).map(str::to_owned).collect();
        visited += step.len();

        let mut next_queue: HashSet<&str> = HashSet::new();
        for node_id in &step {
            if let Some(neighbors) = adjacency.get(node_id.as_str()) {
                for &neighbor in neighbors {
                    let deg = in_degree.get_mut(neighbor).unwrap();
                    *deg -= 1;
                    if *deg == 0 {
                        next_queue.insert(neighbor);
                    }
                }
            }
        }

        steps.push(step);
        queue.extend(next_queue);
    }

    if visited != node_ids.len() {
        return Err(GraphError::Cycle);
    }

    Ok(steps)
}
