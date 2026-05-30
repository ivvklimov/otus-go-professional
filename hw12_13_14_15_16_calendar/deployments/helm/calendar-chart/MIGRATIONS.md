# Применение миграций в Kubernetes

> ⚠️ **Важно**: В текущей версии чарта миграции применяются **вручную**. 

## Предусловия
- Установлен `kubectl` с доступом к кластеру
- В локальной папке `migrations/` есть `.sql`-файлы Goose
- Helm-релиз уже установлен: `helm install otus-calendar ...`

### 1. Удалить pod goose-tmp, если он запущен
```bash
kubectl delete pod goose-tmp -n otus-go-professional --force --grace-period=0
```

### 2. Запустить pod goose-tmp с явным Never (не будет рестартовать)
```bash
kubectl run goose-tmp -n otus-go-professional --image=kukymbr/goose-docker:latest --restart=Never --command -- /bin/sh -c "sleep 3600"
```

### 3. Дождаться готовности
```bash
kubectl wait pod/goose-tmp -n otus-go-professional --for=condition=Ready --timeout=30s
```

### 4. Залить миграции на pod и применить их
```bash
kubectl cp migrations/ goose-tmp:/tmp/migrations -n otus-go-professional
kubectl exec -n otus-go-professional goose-tmp -- goose -dir /tmp/migrations postgres "postgres://user:password@otus-calendar-postgres:5432/calendar?sslmode=disable" up
```

### 5. Удалить временный pod
```bash
kubectl delete pod goose-tmp -n otus-go-professional
```
