# Lab 1.1 — The 60-Second Pod

**Thời lượng:** ~45 phút | **CKAD domain:** Application Design and Build (20%)

## Mục tiêu

- Tạo Pod imperatively (không viết YAML từ đầu) với labels, biến môi trường,
  và resource requests/limits.
- Export manifest bằng `--dry-run=client -o yaml` thay vì gõ tay toàn bộ YAML.
- Verify trạng thái Pod hoàn toàn bằng `kubectl get/describe`, không mở
  editor để "nhìn" lại file — đây là tốc độ cần có trong phòng thi.

## Chuẩn bị

```bash
kubectl create namespace flight-tracker --dry-run=client -o yaml | kubectl apply -f -
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
```

## Nhiệm vụ

1. Tạo một Pod tên `speed-pod`, image `nginx:1.25`, **imperatively** (một
   lệnh `kubectl run`), với:
   - Labels: `app=speed`, `tier=web`, `course=ckad`
   - Biến môi trường: `APP_ENV=production`, `LOG_LEVEL=debug`
2. Pod phải có:
   - `resources.requests`: `cpu: 100m`, `memory: 128Mi`
   - `resources.limits`: `cpu: 250m`, `memory: 256Mi`

   Gợi ý: `kubectl run` **không có** flag `--requests`/`--limits` ở phiên
   bản kubectl hiện tại (bị bỏ từ lâu). Bạn cần export YAML trước rồi thêm
   khối `resources` vào, hoặc dùng `--overrides` (xem phần Bonus trong
   `solution.md` nếu muốn thử cách không đụng tới editor lần nào).
3. Export toàn bộ thành file `manifests/speed-pod.yaml` bằng
   `--dry-run=client -o yaml` — không tạo Pod ngay ở bước này.
4. Sửa file để thêm `resources`, sau đó `apply` để tạo Pod thật.
5. Verify **không mở lại file/editor**, chỉ dùng các lệnh truy vấn:
   - Pod đang `Running`, `1/1` container sẵn sàng.
   - Đúng 3 label.
   - Đúng 2 biến môi trường.
   - Đúng requests/limits như yêu cầu.

   Gợi ý các lệnh nên luyện: `kubectl get pod -o wide`,
   `kubectl get pod --show-labels`, `kubectl describe pod`,
   `kubectl get pod -o jsonpath=...`, `kubectl wait`.

## Tiêu chí hoàn thành

- [ ] `kubectl get pod speed-pod` → `STATUS=Running`, `READY=1/1`
- [ ] `kubectl get pod speed-pod --show-labels` → có đủ `app=speed,tier=web,course=ckad`
- [ ] `kubectl get pod speed-pod -o jsonpath='{.spec.containers[0].resources}'` → khớp requests/limits đề bài
- [ ] Bạn làm xong bước 1–4 (không tính verify) trong **dưới 5 phút**

## Dọn dẹp

```bash
kubectl delete pod speed-pod
```
(Giữ namespace `flight-tracker` lại — lab 1.2–1.4 dùng chung.)
