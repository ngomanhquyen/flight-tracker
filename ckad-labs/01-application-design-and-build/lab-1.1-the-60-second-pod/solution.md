# Lab 1.1 — Solution

Thử làm trước khi đọc phần này.

> Bản thực thi: `bash scripts/solve.sh` chạy toàn bộ lời giải bên dưới và
> tự kiểm tra kết quả (`[PASS]`/`[FAIL]`); `bash scripts/cleanup.sh` để dọn
> dẹp. Lab này không đụng vào resource thật nào.

## Bước 1–3: tạo + export

```bash
kubectl run speed-pod \
  --image=nginx:1.25 \
  --labels="app=speed,tier=web,course=ckad" \
  --env="APP_ENV=production" \
  --env="LOG_LEVEL=debug" \
  --dry-run=client -o yaml > manifests/speed-pod.yaml
```

`--dry-run=client -o yaml` không gọi API server để tạo gì cả — nó chỉ nhờ
`kubectl` build object trong bộ nhớ rồi in ra YAML. Đây là cách nhanh nhất để
có một khung YAML đúng cú pháp mà không phải nhớ toàn bộ schema của Pod.

## Bước 4: thêm resources rồi apply

Mở `manifests/speed-pod.yaml`, thêm `resources` vào container `speed-pod`:

```yaml
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "250m"
        memory: "256Mi"
```

```bash
kubectl apply -f manifests/speed-pod.yaml
```

## Bước 5: verify không mở editor

```bash
kubectl get pod speed-pod -o wide
kubectl wait --for=condition=Ready pod/speed-pod --timeout=30s

kubectl get pod speed-pod --show-labels

kubectl describe pod speed-pod
# hoặc trích riêng phần cần, không đọc cả trang:
kubectl describe pod speed-pod | grep -A6 "Limits\|Requests"

kubectl get pod speed-pod \
  -o jsonpath='{.spec.containers[0].resources}{"\n"}'

kubectl get pod speed-pod \
  -o jsonpath='{range .spec.containers[0].env[*]}{.name}={.value}{"\n"}{end}'
```

`jsonpath` là kỹ năng đáng luyện kỹ cho kỳ thi thật: nhanh hơn `describe`
rất nhiều khi bạn chỉ cần một trường cụ thể để dán vào script kiểm tra hoặc
so sánh nhanh bằng mắt.

## Bonus — 0 lần chạm editor

`--overrides` nhận một đoạn JSON được merge vào object mà `kubectl run` sinh
ra, cho phép set cả `resources` ngay trong một lệnh:

```bash
kubectl run speed-pod-v2 \
  --image=nginx:1.25 \
  --overrides='{
    "metadata": {"labels": {"app":"speed","tier":"web","course":"ckad"}},
    "spec": {
      "containers": [{
        "name": "speed-pod-v2",
        "image": "nginx:1.25",
        "env": [
          {"name":"APP_ENV","value":"production"},
          {"name":"LOG_LEVEL","value":"debug"}
        ],
        "resources": {
          "requests": {"cpu":"100m","memory":"128Mi"},
          "limits": {"cpu":"250m","memory":"256Mi"}
        }
      }]
    }
  }'
```

**Lưu ý quan trọng:** `overrides.spec.containers` là một *strategic merge*
theo kiểu thay thế toàn bộ mảng, không merge từng phần tử. Nếu bạn kết hợp
`--env`/`--labels` (sinh container/metadata mặc định) **cùng lúc** với
`overrides.spec.containers`, phần container do `--overrides` cung cấp sẽ đè
mất `env` sinh ra từ `--env`. Vì vậy ở ví dụ trên, container trong
`overrides` khai báo lại đầy đủ `name`/`image`/`env`/`resources` — không dựa
vào flag nào khác cho phần container. Đây là lỗi rất dễ mắc và mất thời gian
debug trong phòng thi nếu không biết trước.

## Dọn dẹp

```bash
kubectl delete pod speed-pod speed-pod-v2 --ignore-not-found
```
