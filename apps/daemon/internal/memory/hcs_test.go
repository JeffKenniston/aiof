package memory

import (
	"testing"
)

func TestHypervectorBindingSelfInverse(t *testing.T) {
	hv1 := RandomHypervector("concept_a")
	hv2 := RandomHypervector("concept_b")

	// Binding (XOR)
	bound := Bind(hv1, hv2)

	// Self-inverse property: bound ^ hv2 == hv1
	unbound := Bind(bound, hv2)

	sim := Similarity(unbound, hv1)
	if sim != 1.0 {
		t.Errorf("Expected self-inverse binding similarity to be 1.0, got %f", sim)
	}
}

func TestHypervectorOrthogonality(t *testing.T) {
	hv1 := RandomHypervector("concept_alpha")
	hv2 := RandomHypervector("concept_beta")

	sim := Similarity(hv1, hv2)
	// Random 10,000-bit vectors should have similarity around ~0.50 (orthogonal)
	if sim < 0.40 || sim > 0.60 {
		t.Errorf("Expected similarity between random hypervectors to be approx 0.50, got %f", sim)
	}
}

func TestHypervectorBundling(t *testing.T) {
	hv1 := RandomHypervector("concept_x")
	hv2 := RandomHypervector("concept_y")
	hv3 := RandomHypervector("concept_z")

	bundled := Bundle([]Hypervector{hv1, hv2, hv3})

	// Bundled vector should have high similarity (>0.55) to all constituent vectors
	sim1 := Similarity(bundled, hv1)
	sim2 := Similarity(bundled, hv2)
	sim3 := Similarity(bundled, hv3)

	if sim1 <= 0.55 || sim2 <= 0.55 || sim3 <= 0.55 {
		t.Errorf("Expected bundled similarity > 0.55, got sim1=%f, sim2=%f, sim3=%f", sim1, sim2, sim3)
	}
}

func TestPermutation(t *testing.T) {
	hv := RandomHypervector("concept_seq")
	permuted := Permute(hv, 100)

	if Similarity(hv, permuted) > 0.60 {
		t.Errorf("Permuted hypervector should be orthogonal to original")
	}

	unpermuted := Permute(permuted, -100)
	if Similarity(hv, unpermuted) != 1.0 {
		t.Errorf("Inverse permutation failed, expected similarity 1.0, got %f", Similarity(hv, unpermuted))
	}
}

func TestMerkleDAGAndStateRoot(t *testing.T) {
	dag := NewMerkleDAG()

	hv1 := EncodeConcept("golang_backend")
	node1, err := dag.AddNode("node-1", []byte("package main\nfunc main() {}"), hv1, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add node 1: %v", err)
	}

	root1 := dag.GetStateRoot()
	if root1 == "" {
		t.Fatalf("State root should not be empty")
	}

	hv2 := EncodeConcept("python_script")
	node2, err := dag.AddNode("node-2", []byte("print('hello')"), hv2, []string{node1.ID}, nil)
	if err != nil {
		t.Fatalf("Failed to add node 2: %v", err)
	}

	root2 := dag.GetStateRoot()
	if root2 == root1 {
		t.Errorf("State root should update after adding node 2")
	}

	proof, err := dag.GenerateProof(node2.ID)
	if err != nil {
		t.Fatalf("Failed to generate proof for node 2: %v", err)
	}

	if !dag.VerifyProof(proof) {
		t.Errorf("Proof verification failed for node 2")
	}
}

func TestTopKSearch(t *testing.T) {
	dag := NewMerkleDAG()

	hvGo := EncodeConcept("golang_concurrency_goroutine")
	hvPy := EncodeConcept("python_asyncio_event_loop")
	hvRust := EncodeConcept("rust_async_tokio_future")

	_, _ = dag.AddNode("go-node", []byte("go concurrency code"), hvGo, nil, nil)
	_, _ = dag.AddNode("py-node", []byte("python code"), hvPy, nil, nil)
	_, _ = dag.AddNode("rust-node", []byte("rust code"), hvRust, nil, nil)

	queryHV := EncodeConcept("golang_concurrency_goroutine")
	results := dag.TopKSearch(queryHV, 2)

	if len(results) != 2 {
		t.Fatalf("Expected 2 search results, got %d", len(results))
	}

	if results[0].Node.ID != "go-node" {
		t.Errorf("Expected top result to be go-node, got %s", results[0].Node.ID)
	}

	if results[0].Similarity != 1.0 {
		t.Errorf("Expected exact concept match similarity to be 1.0, got %f", results[0].Similarity)
	}
}
