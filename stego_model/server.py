from fastapi import FastAPI
import joblib
import numpy as np
from pydantic import BaseModel

app = FastAPI()

model = joblib.load("stego_model.pkl")

class Features(BaseModel):
    features: list[float]

@app.post("/predict")
def predict(features: Features):
    x = np.array(features.features).reshape(1, -1)
    prob = model.predict_proba(x)[0, 1]  
    return {"stego_prob": float(prob)}
