# Lab 1.3 — Jobs & CronJobs

**Thời lượng:** ~45 phút | **CKAD domain:** Application Design and Build (20%)

## Mục tiêu

- Đọc hiểu một CronJob **thật** đang chạy trong project (`sync-service`) và
  trigger một lần chạy thủ công không cần chờ lịch.
- Chạy một Job cố tình fail và hiểu `backoffLimit`.
- Tạo một CronJob của riêng bạn để quan sát nhiều chu kỳ nhanh.
- Phân biệt rõ Job/CronJob với Deployment: khi nào dùng cái nào, và ai sở
  hữu Pod trong từng trường hợp.

## Chuẩn bị

Dùng lại namespace `flight-tracker`.

```bash
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
```

Phần A cần `sync-service` đã được deploy — nếu bạn đã làm Lab 1.2 Phần A
(`./deploy-local.sh`), nó đã có sẵn trong namespace này (không có
`condition:` gate trong `Chart.yaml`, luôn được cài cùng release). Không
cần deploy gì thêm; không cần dependency nào khác phải "healthy" để làm
bước 1–2 của Phần A.

## Nhiệm vụ

### Phần A — CronJob thật: `sync-service`

`sync-service` (`services/sync-service/`) là một CronJob thật trong chính
project này — poll dữ liệu chuyến bay theo lịch rồi thoát, đúng bản chất
run-to-completion mà cả lab này xoay quanh (xem
[`charts/sync-service/values.yaml`](../../../deployments/helm/flight-tracker/charts/sync-service/values.yaml)
để biết nó được cấu hình thế nào — nhưng làm bài dưới đây bằng cách đọc từ
cluster, không đọc file).

1. Dùng `kubectl get cronjob sync-service -o jsonpath=...` để tự tra ra:
   `schedule`, `concurrencyPolicy`, `backoffLimit` của `jobTemplate`,
   `activeDeadlineSeconds`.
2. Trigger một lần chạy thủ công **ngay lập tức**, không chờ tới chu kỳ kế
   tiếp — có một lệnh `kubectl create job` dành riêng cho việc "chạy ngay
   một CronJob có sẵn" mà không cần định nghĩa lại toàn bộ Pod template.
   Tìm và dùng lệnh đó.
3. Theo dõi Job/Pod vừa tạo (`kubectl get pods -w`). Nó có thể `Complete`
   (nếu Postgres/RabbitMQ/`flight-service` đều sẵn sàng trong cluster của
   bạn) hoặc fail và retry theo `backoffLimit` (nếu thiếu dependency nào) —
   **cả 2 kết quả đều là quan sát hợp lệ** cho bài này, không cần cố ép nó
   thành công.
4. Tự kiểm tra: Job bạn vừa tạo thủ công ở bước 2 có được `sync-service`
   "nhận làm con" (`ownerReferences` trỏ về CronJob) giống các Job tự sinh
   theo lịch không? Dự đoán trước, rồi kiểm tra bằng `kubectl get job
   <tên-job-bạn-tạo> -o jsonpath='{.metadata.ownerReferences}'` — so sánh
   với `ownerReferences` của một Job tự sinh theo lịch
   (`kubectl get jobs -l app.kubernetes.io/name=sync-service`, chọn một cái
   không phải job bạn vừa tạo). Hai kết quả gần giống nhau đến mức nào?
   Khác nhau đúng một field — tìm ra field đó.

### Phần B — Job fail, quan sát backoffLimit

5. Tạo Job tên `fail-job`, image `busybox`, lệnh cố tình fail
   (`sh -c "exit 1"`), với `spec.backoffLimit: 2`.

   Gợi ý: `kubectl create job` không có flag để set `backoffLimit` — export
   YAML trước (`--dry-run=client -o yaml`) rồi thêm field vào.
6. Theo dõi Pod của Job này được tạo lại nhiều lần (retry) do fail liên tục,
   cho tới khi vượt `backoffLimit` thì Job dừng hẳn và được đánh dấu thất
   bại. (Job này tất định — không phụ thuộc trạng thái cluster của bạn như
   Phần A, dùng để đảm bảo bạn luôn quan sát được hành vi backoff dù
   `sync-service` ở Phần A thành công hay thất bại.)

### Phần C — CronJob của riêng bạn

7. Tạo CronJob tên `hello-cron`, imperatively (`kubectl create cronjob`),
   chạy mỗi phút (`*/1 * * * *` — nhanh hơn nhiều so với lịch `*/5 * * * *`
   thật của `sync-service`, để bạn quan sát được vài chu kỳ trong thời
   lượng lab), image `busybox`, in ra ngày giờ hiện tại và dòng chữ
   `Hello from CKAD lab`.
8. Đợi ít nhất 2 chu kỳ (2 phút), quan sát các Job con được CronJob tự sinh
   ra, và log của một trong số chúng.

### Phần D — So sánh

9. Dùng `kubectl explain` để tự tra và điền vào bảng dưới (không tra
   Google): `restartPolicy` được phép của Pod template, ai sở hữu Pod
   (owner chain), và mục đích sử dụng chính.

   | | Job | CronJob | Deployment |
   |---|---|---|---|
   | `restartPolicy` được phép trong Pod template | | | |
   | Owner chain của Pod | | | |
   | Dùng khi nào | | | |

## Tiêu chí hoàn thành

**Phần A:**
- [ ] Bạn tra được đúng 4 giá trị cấu hình của `sync-service` (bước 1) chỉ bằng `kubectl get`/`jsonpath`, không mở file YAML nào
- [ ] Job thủ công ở bước 2 được tạo mà không cần viết YAML từ đầu
- [ ] Bạn trả lời đúng câu hỏi ownership ở bước 4 (kiểm tra bằng lệnh thật, không đoán)

**Phần B–D:**
- [ ] `kubectl get pods -l job-name=fail-job` → thấy nhiều hơn 1 Pod (retry) nhưng không vượt quá `backoffLimit + 1` lần thử
- [ ] `kubectl describe job fail-job` → phần Events có nhắc tới backoff/limit
- [ ] `kubectl get jobs` → sau ≥2 phút thấy ≥2 Job con do `hello-cron` sinh ra, tên dạng `hello-cron-<timestamp>`
- [ ] Bảng so sánh Job/CronJob/Deployment điền đầy đủ

## Dọn dẹp

```bash
kubectl delete job fail-job
kubectl delete job <tên-job-bạn-tạo-ở-Phần-A-bước-2>
kubectl delete cronjob hello-cron   # xoá sớm để tránh spam Job con
```
Không xoá CronJob `sync-service` — đó là một phần của release Helm thật,
không phải resource của lab này.
