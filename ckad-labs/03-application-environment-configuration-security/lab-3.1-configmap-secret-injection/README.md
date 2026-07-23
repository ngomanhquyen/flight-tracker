# Lab 3.1 — ConfigMap & Secret Injection

**Thời lượng:** ~45 phút | **CKAD domain:** Application Environment, Configuration and Security (25%)

## Mục tiêu

- Tạo 1 Secret từ file (`kubectl create secret generic --from-file`) và 1
  ConfigMap từ literal (`--from-literal`).
- Trong **cùng 1 Pod**: inject Secret dưới dạng biến môi trường (chọn đúng
  1 key qua `env.valueFrom.secretKeyRef`), và mount ConfigMap dưới dạng
  volume (mỗi key thành 1 file).
- Đọc hiểu ConfigMap/Secret **thật** của project trước — mọi service hiện
  tại chỉ dùng kiểu `envFrom` (bơm toàn bộ key thành env var), chưa service
  nào mount ConfigMap làm volume — nên phần B là kỹ năng bổ sung, không có
  sẵn trong project để chỉ "đọc lại".

## Chuẩn bị

```bash
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
```

## Phần A — Đọc ConfigMap/Secret thật của `bot-service`

1. `kubectl get configmap bot-service -o yaml` — xem toàn bộ key/value
   plaintext (`BOT_HTTP_PORT`, `BOT_CLIENTS_FLIGHT_SERVICE_URL`...).
2. `kubectl get secret bot-service -o yaml` — xem các key nhạy cảm
   (`BOT_TELEGRAM_BOT_TOKEN`...), thử giải mã 1 giá trị bằng
   `echo '<value>' | base64 -d` để xác nhận đây chỉ là encode, không phải
   mã hoá thật sự.
3. `kubectl get deployment bot-service -o jsonpath='{.spec.template.spec.containers[0].envFrom}'`
   — xác nhận cả 2 object được bơm vào container qua `envFrom`
   (`configMapRef`/`secretRef`, bơm **toàn bộ** key), không phải `env`
   liệt kê từng key một.

## Phần B — Tự tạo Secret từ file + ConfigMap từ literal, inject cả 2 kiểu

1. Tạo 1 file giả lập, tạo Secret từ file đó:

   ```bash
   echo -n "super-secret-api-key" > /tmp/api-key.txt
   kubectl create secret generic demo-secret --from-file=api-key=/tmp/api-key.txt
   ```

2. Tạo ConfigMap từ literal:

   ```bash
   kubectl create configmap demo-config \
     --from-literal=LOG_LEVEL=debug --from-literal=APP_MODE=demo
   ```

3. Viết `manifests/config-demo-pod.yaml`: 1 Pod tên `config-demo-pod`
   (image `busybox:1.36`, lệnh đơn giản kiểu `sleep 3600` để giữ Pod sống
   cho bạn `exec` vào kiểm tra) với:
   - 1 biến môi trường `API_KEY` lấy từ Secret `demo-secret` key `api-key`
     — dùng `env: - name: API_KEY, valueFrom: secretKeyRef: {...}`, **không**
     dùng `envFrom` (khác cách project đang làm, đúng ý luyện phần này).
   - ConfigMap `demo-config` mount làm volume tại `/etc/demo-config`.

4. Apply, rồi kiểm tra:

   ```bash
   kubectl apply -f manifests/config-demo-pod.yaml
   kubectl exec config-demo-pod -- printenv API_KEY
   kubectl exec config-demo-pod -- ls /etc/demo-config
   kubectl exec config-demo-pod -- cat /etc/demo-config/LOG_LEVEL
   ```

## Tiêu chí hoàn thành

- [ ] `kubectl get secret demo-secret -o jsonpath='{.data.api-key}' | base64 -d` → in đúng `super-secret-api-key`
- [ ] `kubectl exec config-demo-pod -- printenv API_KEY` → in đúng giá trị Secret
- [ ] `kubectl exec config-demo-pod -- ls /etc/demo-config` → thấy 2 file `LOG_LEVEL`, `APP_MODE`
- [ ] `kubectl exec config-demo-pod -- cat /etc/demo-config/LOG_LEVEL` → in `debug`

## Dọn dẹp

```bash
kubectl delete pod config-demo-pod --ignore-not-found
kubectl delete secret demo-secret --ignore-not-found
kubectl delete configmap demo-config --ignore-not-found
rm -f /tmp/api-key.txt
```

(Không đụng gì tới ConfigMap/Secret thật của `bot-service` — Phần A chỉ đọc.)
