import json
import random
import os
import argparse
import sys

try:
    import torch
    import torch.nn as nn
    import torch.optim as optim
except ImportError:
    print("Error: PyTorch is not installed. Please run `pip install torch`")
    sys.exit(1)

class RouteLLMClassifier(nn.Module):
    """
    A Multi-Layer Perceptron (MLP) for predicting prompt complexity
    from 768-dimensional text embeddings.
    """
    def __init__(self, embedding_dim=768, hidden_dim=128):
        super().__init__()
        # Layer 1: Project embedding to hidden dimension
        self.fc1 = nn.Linear(embedding_dim, hidden_dim)
        # Activation: ReLU
        self.relu = nn.ReLU()
        # Layer 2: Output a single complexity score
        self.fc2 = nn.Linear(hidden_dim, 1)
        
    def forward(self, x):
        x = self.fc1(x)
        x = self.relu(x)
        x = self.fc2(x)
        return x

def generate_synthetic_data(num_samples=1000, embedding_dim=768):
    """
    Generates synthetic training data. In a real environment, this would
    load historical prompt logs and APGR (Average Performance Gap Recovered) scores.
    """
    print(f"Generating {num_samples} synthetic training samples...")
    X = torch.randn(num_samples, embedding_dim)
    
    # Create a synthetic target: if the sum of the first 10 dimensions is > 0, 
    # it's complex (1.0), else simple (0.0), adding some noise.
    y = (X[:, :10].sum(dim=1) > 0).float().unsqueeze(1)
    
    return X, y

def export_weights_to_json(model, filepath):
    """
    Exports the trained weights and biases to a flat JSON structure
    compatible with the Go backend's two-pass deserializer.
    """
    print(f"Exporting calibrated weights to {filepath}...")
    state = model.state_dict()
    weights = {
        "fc1.weight": state["fc1.weight"].tolist(),
        "fc1.bias": state["fc1.bias"].tolist(),
        "fc2.weight": state["fc2.weight"].tolist(),
        "fc2.bias": state["fc2.bias"].tolist()
    }
    
    os.makedirs(os.path.dirname(filepath), exist_ok=True)
    with open(filepath, 'w') as f:
        json.dump(weights, f, indent=2)
    print(f"Export complete. File size: {os.path.getsize(filepath)} bytes.")

def main():
    parser = argparse.ArgumentParser(description="AIOF RouteLLM Offline Calibrator")
    parser.add_argument("--epochs", type=int, default=50, help="Number of training epochs")
    parser.add_argument("--samples", type=int, default=5000, help="Number of synthetic samples")
    parser.add_argument("--output", type=string, default="../route_llm_weights.json", help="Output path for weights")
    
    args = parser.parse_args()
    
    X, y = generate_synthetic_data(num_samples=args.samples)
    
    model = RouteLLMClassifier()
    criterion = nn.MSELoss()
    optimizer = optim.Adam(model.parameters(), lr=0.01)
    
    print("Starting Matrix Factorization Calibration...")
    for epoch in range(args.epochs):
        optimizer.zero_grad()
        outputs = model(X)
        loss = criterion(outputs, y)
        loss.backward()
        optimizer.step()
        
        if (epoch + 1) % 10 == 0:
            print(f"Epoch [{epoch+1}/{args.epochs}], Loss: {loss.item():.4f}")
            
    print("Calibration finished successfully.")
    
    # Export weights
    out_path = os.path.abspath(os.path.join(os.path.dirname(__file__), args.output))
    export_weights_to_json(model, out_path)

if __name__ == "__main__":
    # Fix the argparse string type issue
    import sys
    # Re-declare parser correctly
    parser = argparse.ArgumentParser(description="AIOF RouteLLM Offline Calibrator")
    parser.add_argument("--epochs", type=int, default=50, help="Number of training epochs")
    parser.add_argument("--samples", type=int, default=5000, help="Number of synthetic samples")
    parser.add_argument("--output", type=str, default="../route_llm_weights.json", help="Output path for weights")
    
    args = parser.parse_args()
    
    X, y = generate_synthetic_data(num_samples=args.samples)
    
    model = RouteLLMClassifier()
    criterion = nn.MSELoss()
    optimizer = optim.Adam(model.parameters(), lr=0.01)
    
    print("Starting Matrix Factorization Calibration...")
    for epoch in range(args.epochs):
        optimizer.zero_grad()
        outputs = model(X)
        loss = criterion(outputs, y)
        loss.backward()
        optimizer.step()
        
        if (epoch + 1) % 10 == 0:
            print(f"Epoch [{epoch+1}/{args.epochs}], Loss: {loss.item():.4f}")
            
    print("Calibration finished successfully.")
    
    out_path = os.path.abspath(os.path.join(os.path.dirname(__file__), args.output))
    export_weights_to_json(model, out_path)
