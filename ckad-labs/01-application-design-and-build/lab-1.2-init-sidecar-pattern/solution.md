# Lab 1.2 — Solution

Thử làm trước khi đọc phần này.

> Bản thực thi: `bash scripts/solve.sh` chạy Phần A (đọc bot-service thật,
> bỏ qua với cảnh báo nếu chưa deploy) và Phần B (tạo `log-pipeline`), kèm
> tự kiểm tra `[PASS]`/`[FAIL]`. `bash scripts/cleanup.sh` chỉ xoá
> `log-pipeline` — Phần A không tạo/sửa gì nên không cần dọn.

## Phần A — Sidecar thật: `bot-service`

```bash
kubectl get pods -l app.kubernetes.io/name=bot-service
# lấy tên pod, gọi $POD từ đây trở đi
POD=$(kubectl get pods -l app.kubernetes.io/name=bot-service -o jsonpath='{.items[0].metadata.name}')

kubectl get pod "$POD" -o jsonpath='{.spec.containers[*].name}{"\n"}'
# bot-service cloudflared webhook-registrar
```

```bash
kubectl get pod "$POD" -o jsonpath='{.spec.volumes[*].name}{"\n"}'
# tunnel-log registrar-script kube-api-access-...
kubectl get pod "$POD" -o jsonpath='{.spec.volumes[?(@.name=="tunnel-log")]}{"\n"}'
# {"emptyDir":{},"name":"tunnel-log"}
```

```bash
kubectl logs "$POD" -c cloudflared --tail=20
kubectl logs "$POD" -c webhook-registrar -f
# Ctrl+C sau vài giây — thấy dòng "webhook-registrar: watching ... for a
# trycloudflare.com URL" lặp lại mỗi 5s (từ vòng `sleep 5` trong script)
```

```bash
kubectl exec "$POD" -c webhook-registrar -- cat /shared/cloudflared.log
```

Nếu lệnh trên đọc được nội dung (dù chỉ là log kết nối của cloudflared, có
thể chưa có URL nếu tunnel chưa lên hẳn), nghĩa là 2 container đang thật sự
đọc/ghi chung filesystem qua volume `tunnel-log` — đúng cơ chế `emptyDir`
bạn tự dựng ở Phần B, chỉ khác là ở đây `cloudflared` ghi và
`webhook-registrar` đọc, thay vì `app` ghi và `log-shipper` đọc.

**Vì sao `bot-service` (container chính) không mount `tunnel-log`:** nó
hoàn toàn không cần biết tunnel tồn tại — nó chỉ lắng nghe HTTP trên
`localhost:8080` như bình thường. `cloudflared` trỏ vào cổng đó từ bên
ngoài, và `webhook-registrar` chỉ nói chuyện với Telegram API, không bao
giờ gọi vào `bot-service`. Đây chính là điểm mạnh của sidecar pattern: mối
quan tâm phụ (expose Pod ra Internet khi không có Ingress) được cô lập
hoàn toàn khỏi code nghiệp vụ chính — xoá 2 sidecar này khỏi
`extraContainers` không cần sửa một dòng Go nào trong `bot-service`.

## Phần B — Tự viết init container

`manifests/log-pipeline.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: log-pipeline
  labels:
    app: log-pipeline
spec:
  volumes:
    - name: shared-logs
      emptyDir: {}
  initContainers:
    - name: init-setup
      image: busybox:1.36
      command:
        - sh
        - -c
        - echo "Init completed at $(date)" | tee /var/log/app/init.log
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/app
  containers:
    - name: app
      image: busybox:1.36
      command:
        - sh
        - -c
        - |
          while true; do
            echo "App tick $(date)" >> /var/log/app/app.log
            sleep 5
          done
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/app
    - name: log-shipper
      image: busybox:1.36
      command:
        - sh
        - -c
        - touch /var/log/app/app.log && tail -n+1 -f /var/log/app/app.log
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/app
```

```bash
kubectl apply -f manifests/log-pipeline.yaml
```

## Điểm cần hiểu, không chỉ chép lệnh

- **`emptyDir` sống theo vòng đời Pod**, không theo container — đúng cơ chế
  bạn vừa thấy `bot-service` dùng thật ở Phần A, chỉ khác chiều đọc/ghi.
- **Init container chạy tuần tự và chạy xong trước khi bất kỳ container
  thường nào start.** Nhưng `app` và `log-shipper` (2 container thường)
  **khởi động song song, không đảm bảo thứ tự** — giống `cloudflared` và
  `webhook-registrar` ở Phần A cũng khởi động song song với nhau. Đây là lý
  do lệnh của `log-shipper` có `touch /var/log/app/app.log &&` trước
  `tail -f`: nếu nó start trước khi `app` kịp ghi dòng đầu tiên, `tail -f`
  trên file chưa tồn tại sẽ khiến container lỗi/exit. `touch` đảm bảo file
  luôn tồn tại trước khi tail, bất kể thứ tự start thực tế.
- `READY` của Pod hiển thị `2/2`, không phải `3/3` — init container không
  được tính vào readiness count của Pod đang chạy, nó chỉ có ý nghĩa trong
  giai đoạn khởi động (`Init:x/y`).
- **`kubectl logs` chỉ đọc stdout/stderr của container, không đọc file.**
  Lệnh của `init-setup` dùng `echo ... | tee /var/log/app/init.log` thay vì
  `echo ... >> /var/log/app/init.log` — `tee` vừa ghi ra file (để `app` đọc
  qua volume chung) vừa in ra stdout (để `kubectl logs -c init-setup` thấy
  được dòng đó). Chỉ dùng `>>` (redirect thuần) thì log container rỗng dù
  file vẫn được ghi đúng — dễ nhầm tưởng container "không làm gì" trong khi
  thực ra nó làm đúng việc, chỉ là làm âm thầm.
- **Vì sao `log-pipeline` cần init container mà `bot-service` thì không:**
  ở đây là bài tập giả định container chính *phụ thuộc dữ liệu* do init
  container chuẩn bị trước (dù trong ví dụ này không thực sự blocking gì
  quan trọng). `bot-service` thì ngược lại — nó chủ động chấp nhận chạy ở
  trạng thái "degraded" khi dependency (`flight-service`/
  `subscription-service`) chưa sẵn sàng, tự phục hồi khi dependency online,
  thay vì bị k8s giữ ở `Init:` chờ vô thời hạn nếu dependency đó chậm lên.
  Init container là lựa chọn đúng khi **thiếu điều kiện tiên quyết đó thì
  container chính không thể chạy đúng**; sai lựa chọn khi container chính
  vẫn có thể chạy — chỉ ở chế độ hạn chế — mà không cần điều kiện đó.

## Verify

```bash
kubectl get pod log-pipeline -w
# Ctrl+C khi thấy Running   2/2

kubectl get pod log-pipeline \
  -o jsonpath='{.status.initContainerStatuses[0].state}{"\n"}'

kubectl logs log-pipeline -c init-setup

kubectl logs log-pipeline -c log-shipper --tail=5 -f
# Ctrl+C sau vài giây, quan sát dòng mới xuất hiện mỗi ~5s

kubectl exec log-pipeline -c app -- cat /var/log/app/init.log
```

## Dọn dẹp

```bash
kubectl delete pod log-pipeline
```
