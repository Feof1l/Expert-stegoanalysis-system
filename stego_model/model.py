import pandas as pd
from sklearn.model_selection import train_test_split
from sklearn.neural_network import MLPClassifier

df = pd.read_csv("../stego_analyzer/features.csv")

# Бинарная метка: 0 = clean, 1 = stego
df["y"] = (df["label"] > 0).astype(int)

X = df[[
    "lsb_transitions",
    "bit_run_00", "bit_run_01", "bit_run_10", "bit_run_11",
    "neighbor_diff",
    "chi_square",
    "entropy_lsb",
    "r", "s", "rm", "sm", "rm_r", "sm_s",
]]

y = df["y"]

X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

model = MLPClassifier(hidden_layer_sizes=(32, 16), max_iter=1000, random_state=42)
model.fit(X_train, y_train)

print("Test accuracy:", model.score(X_test, y_test))

# Сохранить модель
import joblib
joblib.dump(model, "stego_model.pkl")
