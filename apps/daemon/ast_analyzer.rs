use std::cmp::Ordering;
use std::sync::{Arc, Mutex};
use std::thread;

/// An AST Node representing the parsed code
#[derive(Debug)]
pub struct AstNode {
    pub id: usize,
    pub value: String,
}

/// A thread-safe wrapper around an AST Node
pub type SharedAstNode = Arc<Mutex<AstNode>>;

/// Compare two AST trees to compute their diff
/// Here, we just check if they are identical in their current state.
pub fn analyze_ast_diff(node_a: &SharedAstNode, node_b: &SharedAstNode) -> bool {
    // INVARIANT SOLVED: To prevent deadlocks, always lock resources in a globally consistent order.
    // We enforce locking the node with the lower ID first.
    
    let id_a = node_a.lock().unwrap().id;
    let id_b = node_b.lock().unwrap().id;
    
    let (lock1, lock2) = match id_a.cmp(&id_b) {
        Ordering::Less => (node_a.lock().unwrap(), node_b.lock().unwrap()),
        Ordering::Greater => {
            // Lock B first, then A
            let lb = node_b.lock().unwrap();
            let la = node_a.lock().unwrap();
            (la, lb)
        },
        Ordering::Equal => {
            // They are the exact same node ID, or same node. 
            // Just lock one to compare to itself.
            let la = node_a.lock().unwrap();
            return true;
        }
    };
    
    lock1.value == lock2.value
}

pub fn main() {
    let node1 = Arc::new(Mutex::new(AstNode { id: 1, value: "Node1".into() }));
    let node2 = Arc::new(Mutex::new(AstNode { id: 2, value: "Node2".into() }));
    
    let is_same = analyze_ast_diff(&node1, &node2);
    println!("AST Diff identical: {}", is_same);
}
