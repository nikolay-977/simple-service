
## Клонируйте репозиторий:

```bash
git clone https://github.com/nikolay-977/go-simple-highload-service
cd go-simple-highload-service
```

# Запуск приложения в Docker
## Обновите зависимости
```bash
go mod tidy
```

## Запустите в Docker
```bash
docker-compose up
```

# Запуск приложения в Minikube

## Запуск в Minikube
```bash
minikube start --driver=docker --cpus=2 --memory=4g
minikube addons enable ingress
minikube addons enable metrics-server
```
## Проверка статуса Minikube
```bash
minikube status
```

## Сборка Docker образа для Minikube
```bash
eval $(minikube docker-env)
docker build -t simple-service:latest .
```

# Примените конфигурации
```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/prometheus.yaml
kubectl apply -f k8s/grafana.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/ingress.yaml
kubectl apply -f k8s/hpa.yaml
```

# Проверьте статус
```bash
kubectl get pods -n monitoring
kubectl get svc -n monitoring
kubectl get pods
kubectl get services
kubectl get ingress
```

# Пробросьте порты
```bash
kubectl port-forward svc/simple-service 8080:8080 &
kubectl port-forward svc/prometheus -n monitoring 9090:9090 &
kubectl port-forward svc/grafana -n monitoring 3000:3000 &
```

# Откройте Prometheus и Grafana
```bash
open http://localhost:9090
open http://localhost:3000
```

# Имортируйте dashboard для Grafana
- [Simple Service - Monitoring & Alerts.json](Simple%20Service%20-%20Monitoring%20%26%20Alerts.json)

# Используйте команду для мониторина масштабирования
```bash
watch -n 1 'kubectl get pods,svc,hpa'
```

# Запустите нагрузочный тест
```bash
locust -f locustfile.py --headless -u 150 -r 10 --run-time 300s --host http://localhost:8080 \
  --csv=locust_results \
  --csv-full-history \
  --html=locust_report.html \
  --logfile=locust.log \
  --loglevel=INFO
```
