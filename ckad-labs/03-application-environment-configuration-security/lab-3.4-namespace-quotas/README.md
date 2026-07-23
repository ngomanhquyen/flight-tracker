# Lab 3.4 — Namespace Quotas

**Thời lượng:** ~45 phút | **CKAD domain:** Application Environment, Configuration and Security (25%)

## Mục tiêu

- Áp `ResourceQuota` (giới hạn tổng `requests`/`limits` cpu+memory ở cấp
  namespace) và `LimitRange` (giới hạn min/max/default cho từng container)
  lên **namespace `flight-tracker` đang chạy thật** (không phải namespace
  rỗng tạo riêng cho lab).
- Quan sát Pod bị `Pending` khi vượt quota — và hiểu vì sao `ResourceQuota`
  khai `requests.cpu`/`requests.memory` sẽ bắt buộc **mọi** container mới
  tạo trong namespace phải khai `resources.requests`.

## Vì sao lab này nối tiếp trực tiếp lab 2.3

Ở lab 2.3, HPA từng báo `<unknown>` mãi vì sidecar `cloudflared`/
`webhook-registrar` (trong `values-local.yaml`, dùng cho `bot-service`
local) thiếu `resources.requests` — đã fix bằng cách thêm
`{cpu: 100m, memory: 100Mi}` cho cả 2 (và cho initContainer `wait-for-db`
trong `common.initContainers`). Nếu bug đó **chưa được fix**, bật
`ResourceQuota` có `requests.cpu`/`requests.memory` lên namespace này bây
giờ sẽ khiến hậu quả nặng hơn nhiều so với "HPA bị mù": Kubernetes sẽ
**từ chối tạo Pod `bot-service` hoàn toàn** (báo lỗi ngay lúc apply, dạng
`forbidden: failed quota: ...: must specify cpu for cloudflared`), vì thiếu
`requests` ở bất kỳ container nào cũng khiến Pod không đủ điều kiện tạo khi
namespace có quota cho field đó. Lab này minh hoạ đúng lý do gotcha ở lab
2.3 quan trọng hơn phạm vi ban đầu tưởng.

## Chuẩn bị

```bash
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
kubectl describe nodes | grep -A5 "Allocated resources"
kubectl top pods -n flight-tracker   # cần metrics-server đã cài ở lab 2.3
```

## Nhiệm vụ

1. Viết `manifests/quota.yaml` — `ResourceQuota` đặt tổng
   `requests.cpu`/`requests.memory`/`limits.cpu`/`limits.memory` cho
   namespace. Chọn con số **thấp hơn 1 chút** so với tổng hiện đang chạy
   (`bot-service` + `flight-service` + `sync-service` + `postgresql` +
   `redis` + `rabbitmq`) — đủ để mọi thứ đang chạy tiếp tục hoạt động bình
   thường, nhưng không đủ dư để scale thêm nhiều.
2. Viết `manifests/limitrange.yaml` — `LimitRange` đặt `min`/`max`/
   `default`/`defaultRequest` cho cpu/memory ở cấp container.
3. Apply cả 2:

   ```bash
   kubectl apply -f manifests/quota.yaml
   kubectl apply -f manifests/limitrange.yaml
   kubectl get pods -n flight-tracker
   ```

   Xác nhận resource hiện có **không bị ảnh hưởng gì** — `ResourceQuota`
   chỉ chặn tạo/sửa Pod mới vượt quota, không evict Pod đang chạy.

4. Lặp lại đúng lệnh scale ở lab 2.3:

   ```bash
   kubectl scale deployment bot-service --replicas=10 -n flight-tracker
   kubectl get pods -l app.kubernetes.io/name=bot-service -n flight-tracker
   ```

   Quan sát: 1 số Pod mới bị kẹt `Pending` — khác cả 2 kiểu lỗi đã gặp ở
   lab 2.1 (`ImagePullBackOff`) và lab 2.2 (`Connection refused`), đây là
   kiểu thứ 3: Pod **không được scheduler nhận** ngay từ đầu.

5. `kubectl describe pod <tên Pod đang Pending>` — xem Event, sẽ có dòng
   nhắc tới `exceeded quota`.

6. `kubectl describe resourcequota` — xem cột `Used`/`Hard` để biết chính
   xác đang chạm giới hạn nào.

## Tiêu chí hoàn thành

- [ ] `kubectl get resourcequota` → thấy quota đã tạo, cột `Used` phản ánh đúng tổng hiện tại
- [ ] Sau khi scale `bot-service` lên 10, `kubectl get pods -l app.kubernetes.io/name=bot-service` → có Pod ở trạng thái `Pending`
- [ ] `kubectl describe pod <Pod Pending>` → Event có nhắc tới `exceeded quota`
- [ ] Scale `bot-service` về lại 1, `kubectl get resourcequota` → `Used` giảm tương ứng

## Dọn dẹp

```bash
kubectl scale deployment bot-service --replicas=1 -n flight-tracker
kubectl delete resourcequota --all -n flight-tracker
kubectl delete limitrange --all -n flight-tracker
```

(Hoặc chạy lại `./deploy-local.sh` từ `deployments/helm/flight-tracker` để
đồng bộ lại toàn bộ namespace theo đúng `values.yaml`/`values-local.yaml`.)
