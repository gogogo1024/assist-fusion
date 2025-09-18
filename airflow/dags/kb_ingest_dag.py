"""
Airflow DAG (draft): Knowledge base ingestion pipeline.

This is a documentation-first stub; adapt paths and connection details before running in a real Airflow env.
"""
from __future__ import annotations

import os
from datetime import datetime, timedelta

try:
	# Airflow imports are optional here; if not available, this file serves as docs.
	from airflow import DAG  # type: ignore
	from airflow.operators.python import PythonOperator  # type: ignore
	exists_airflow = True
except Exception:  # pragma: no cover
	exists_airflow = False
	DAG = object  # type: ignore
	PythonOperator = object  # type: ignore

DATA_DIR = os.getenv("KB_DATA_DIR", os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "data", "docs")))
KB_SVC = os.getenv("KB_SVC", "http://localhost:8082")
AI_SVC = os.getenv("AI_SVC", "http://localhost:8083")


def list_documents(dir_path: str) -> list[str]:
	files: list[str] = []
	for root, _, filenames in os.walk(dir_path):
		for fn in filenames:
			if fn.endswith(('.md', '.txt')):
				files.append(os.path.join(root, fn))
	return files


def slice_text(text: str, max_len: int = 800) -> list[str]:
	# naive splitter by paragraphs
	paras = [p.strip() for p in text.split("\n\n") if p.strip()]
	chunks: list[str] = []
	buf = ""
	for p in paras:
		if len(buf) + len(p) + 2 <= max_len:
			buf = (buf + "\n\n" + p) if buf else p
		else:
			if buf:
				chunks.append(buf)
			buf = p
	if buf:
		chunks.append(buf)
	return chunks


def ingest_doc(filepath: str) -> None:
	# Pseudocode: read -> slice -> (optional) embeddings -> post to kb-svc
	with open(filepath, 'r', encoding='utf-8') as f:
		content = f.read()
	
	title = os.path.basename(filepath)
	chunks = slice_text(content)
	
	# Optional embedding step (pseudo, to keep this file runnable without deps)
	_ = chunks  # normally we would call AI_SVC /embeddings here
	
	# Hand over to kb-svc (pseudo)
	# requests.post(f"{KB_SVC}/v1/docs", json={"title": title, "content": content, "meta": {"chunks": len(chunks)}})
	print(f"[kb-ingest] would ingest {title} with {len(chunks)} chunks → {KB_SVC}")


def ingest_all() -> None:
	files = list_documents(DATA_DIR)
	for fp in files:
		ingest_doc(fp)


def build_dag() -> DAG:
	default_args = {
		"owner": "kb",
		"retries": 0,
		"retry_delay": timedelta(minutes=5),
	}
	with DAG(
		dag_id="kb_ingest_dag",
		start_date=datetime(2024, 1, 1),
		schedule_interval=None,
		catchup=False,
		default_args=default_args,
		description="Knowledge base ingestion (draft)",
	) as dag:
		PythonOperator(task_id="ingest_all", python_callable=ingest_all)
	return dag


if exists_airflow:  # pragma: no cover
	dag = build_dag()
