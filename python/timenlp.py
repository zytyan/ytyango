import time

import fastapi
from jionlp.algorithm.ner.time_extractor import TimeExtractor
from pydantic import BaseModel

app = fastapi.FastAPI()
extractor = TimeExtractor()


class Item(BaseModel):
    text: str


@app.post("/parse_time")
def parse_time(text: Item):
    return extractor(text.text, time_base=time.time())
