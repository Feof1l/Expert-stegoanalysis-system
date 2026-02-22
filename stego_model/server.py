# api.py
from fastapi import FastAPI
import joblib
import numpy as np

app = FastAPI()
model = joblib.load("../stego_model/stego_model.pkl")

@app.post("/predict")
def predict(features: list[float]):
    x = np.array(features).reshape(1, -1)
    prob = model.predict_proba(x)[0, 1]  # вероятность класса 1 (stego)
    return {"stego_prob": float(prob)}
