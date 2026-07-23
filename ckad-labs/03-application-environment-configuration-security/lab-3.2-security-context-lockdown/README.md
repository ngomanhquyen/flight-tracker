# Lab 3.2 — Security Context Lockdown

**Thời lượng:** ~45 phút | **CKAD domain:** Application Environment, Configuration and Security (25%)

## Mục tiêu

- Cấu hình 1 Pod chạy non-root, root filesystem read-only, drop toàn bộ
  Linux capabilities, tắt privilege escalation — bộ `securityContext`
  "khoá chặt" tiêu chuẩn hay gặp trong đề thi CKAD.
- Đối chiếu với thực tế: `common.deployment`/`common.cronjob`
  (`deployments/helm/flight-tracker/charts/common/templates/_helpers.tpl`)
  hiện **không set `securityContext` nào** cho bất kỳ service thật nào —
  dù Dockerfile của các service Go (ví dụ `services/bot-service/Dockerfile`)
  đã dùng base image `gcr.io/distroless/static-debian12:nonroot` với
  `USER nonroot:nonroot`. Nghĩa là non-root hiện chỉ là hệ quả **tình cờ**
  của image, chưa phải điều Kubernetes chủ động enforce — không có gì chặn
  nếu sau này lỡ đổi sang 1 image chạy root, và root filesystem/capabilities
  hiện hoàn toàn không bị giới hạn gì.

## Chuẩn bị

```bash
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
```

## Nhiệm vụ

Viết `manifests/locked-down-pod.yaml`: 1 Pod tên `locked-down-pod` (image
`busybox:1.36`, lệnh `sleep 3600`) với:

1. `spec.securityContext.runAsNonRoot: true` + `runAsUser`/`runAsGroup`
   (chọn 1 UID != 0, ví dụ `1000`).
2. Container-level `securityContext`:
   - `readOnlyRootFilesystem: true`
   - `allowPrivilegeEscalation: false`
   - `capabilities: { drop: ["ALL"] }`
3. Nếu image cần ghi 1 chỗ nào đó (busybox thường không cần), thêm 1
   `emptyDir` volume mount đúng chỗ cần ghi — minh hoạ đúng thực tế:
   `readOnlyRootFilesystem` không cấm ghi vào volume mount riêng, chỉ cấm
   ghi vào filesystem gốc của chính image.

Sau khi apply:

4. `kubectl exec locked-down-pod -- id` — xác nhận UID/GID đúng như đã
   khai, không phải `0`.
5. Thử ghi file vào 1 đường dẫn **không** có volume mount riêng (ví dụ
   `kubectl exec locked-down-pod -- touch /etc/test`) — phải lỗi
   `Read-only file system`.
6. Thử 1 lệnh cần capability đã bị drop (ví dụ `kubectl exec
   locked-down-pod -- ping -c1 8.8.8.8`, cần `CAP_NET_RAW`) — phải bị từ
   chối do `capabilities.drop: ["ALL"]`.

## Thảo luận (chưa quyết định, không bắt buộc làm)

Có nên áp `securityContext` này thẳng vào `common.deployment`/
`common.cronjob` cho tất cả service thật, biến đây thành 1 cải tiến thật
cho project (giống cách `podAnnotations`/initContainer từng được thêm)?
Cần kiểm tra trước: các sidecar `wait-for-db`/`log-shipper` (busybox) và
`cloudflared`/`webhook-registrar` (image bên thứ ba, chỉ dùng ở
`values-local.yaml`) có tương thích non-root/read-only-rootfs không, và
container chính của từng service có ghi gì ra ngoài các volume đã mount rõ
ràng hay không. Việc này tách biệt khỏi phần luyện thi ở trên — bàn riêng
và làm thành 1 việc riêng khi cần.

## Tiêu chí hoàn thành

- [ ] `kubectl exec locked-down-pod -- id` → UID/GID khác `0`
- [ ] Ghi thử file vào đường dẫn không có volume mount → lỗi `Read-only file system`
- [ ] Thử lệnh cần capability đã bị drop → bị từ chối (`Operation not permitted`)

## Dọn dẹp

```bash
kubectl delete pod locked-down-pod --ignore-not-found
```
