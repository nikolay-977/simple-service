from locust import HttpUser, task, between
import random
import json
from datetime import datetime, timezone

class MetricsUser(HttpUser):
    wait_time = between(0.05, 0.1)

    def on_start(self):
        self.total_requests = 0

    @task(100)
    def send_metric(self):
        # Генерация нормальных данных
        timestamp = datetime.now(timezone.utc).isoformat()

        # Базовые значения в нормальном диапазоне
        base_cpu = 30.0
        base_rps = 50.0

        # Нормальный случайный шум
        cpu_noise = random.uniform(-5, 5)
        rps_noise = random.uniform(-10, 10)

        metric = {
            "timestamp": timestamp,
            "cpu": max(0.1, base_cpu + cpu_noise),
            "rps": max(1.0, base_rps + rps_noise)
        }

        headers = {'Content-Type': 'application/json'}

        with self.client.post("/metrics",
                             data=json.dumps(metric),
                             headers=headers,
                             catch_response=True) as response:
            self.total_requests += 1

            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Status: {response.status_code}")

    @task(2)
    def get_analytics(self):
        self.client.get("/analytics")

    @task(1)
    def health_check(self):
        self.client.get("/health")