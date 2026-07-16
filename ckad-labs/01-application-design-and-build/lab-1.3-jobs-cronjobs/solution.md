# Lab 1.3 — Solution

Thử làm trước khi đọc phần này.

> Bản thực thi: `bash scripts/solve.sh` chạy cả 4 phần (Phần A bỏ qua với
> cảnh báo nếu `sync-service` chưa deploy), kèm tự kiểm tra
> `[PASS]`/`[FAIL]`. Phần C (`fail-job`) và Phần A (chờ job thủ công) có
> đợi thật (backoff ~10-40s, CronJob 1 phút/chu kỳ) nên script chạy vài
> phút, không phải vài giây. `bash scripts/cleanup.sh` để dọn dẹp —
> không đụng tới CronJob `sync-service` thật.

## Phần A — CronJob thật: `sync-service`

```bash
kubectl get cronjob sync-service -o jsonpath='{.spec.schedule}{"\n"}'
# */5 * * * *

kubectl get cronjob sync-service -o jsonpath='{.spec.concurrencyPolicy}{"\n"}'
# Forbid

kubectl get cronjob sync-service -o jsonpath='{.spec.jobTemplate.spec.backoffLimit}{"\n"}'
# 2

kubectl get cronjob sync-service -o jsonpath='{.spec.jobTemplate.spec.activeDeadlineSeconds}{"\n"}'
# 240
```

Trigger một lần chạy thủ công, không chờ lịch:

```bash
kubectl create job sync-manual-1 --from=cronjob/sync-service
kubectl get pods -l job-name=sync-manual-1 -w
```

`--from=cronjob/<tên>` sao chép nguyên `jobTemplate` của CronJob (image,
env, resources, `backoffLimit`...) vào một Job mới, tạo ngay lập tức —
đúng công cụ cho "chạy thử một CronJob mà không sửa lịch hay chờ tới chu kỳ
sau", một thao tác rất hay gặp trong thực tế lẫn trong đề thi CKAD.

Nếu Postgres/RabbitMQ/`flight-service` đều sẵn sàng trong cluster của bạn,
Job này `Complete`; nếu thiếu cái nào, nó fail và bạn sẽ thấy đúng hành vi
retry-theo-`backoffLimit` giống hệt Phần B nhưng trên một Job thật thay vì
Job giả lập — `kubectl logs job/sync-manual-1` sẽ cho biết chính xác nó
đang thiếu dependency nào (log lỗi kết nối DB/RabbitMQ/HTTP).

**Câu hỏi ownership (bước 4):** trực giác dễ đoán sai theo cả 2 hướng — kết
quả thật (đã kiểm chứng trên cluster, kubectl v1.36) là:

```bash
kubectl get job sync-manual-1 -o jsonpath='{.metadata.ownerReferences}{"\n"}'
# [{"apiVersion":"batch/v1","controller":true,"kind":"CronJob","name":"sync-service","uid":"..."}]
```

`kubectl create job --from=cronjob/...` **thật sự thiết lập
`ownerReferences`** trỏ về CronJob, `controller: true` — gần như y hệt một
Job tự sinh theo lịch. So sánh trực tiếp với một Job tự sinh
(`kubectl get jobs -l app.kubernetes.io/name=sync-service -o jsonpath=...`
trên một cái không phải `sync-manual-1`), khác biệt duy nhất là Job tự sinh
có thêm `"blockOwnerDeletion":true` mà `sync-manual-1` không có — một field
phụ, không đổi bản chất "là con của CronJob" của cả hai.

**Hệ quả thực tế, dễ gây bất ngờ:** vì `sync-manual-1` đã được `sync-service`
"nhận làm con", `kubectl delete cronjob sync-service` sẽ **cascade-xoá cả
nó** chứ không chỉ các Job tự sinh theo lịch — khác với trực giác "job tôi
tự tạo thủ công thì tôi tự quản lý, xoá CronJob không đụng tới nó". Muốn
xoá CronJob mà giữ lại các Job đã sinh ra (thủ công lẫn theo lịch), dùng
`kubectl delete cronjob sync-service --cascade=orphan`.

## Phần B — Job fail + backoffLimit

```bash
kubectl create job fail-job --image=busybox \
  --dry-run=client -o yaml \
  -- sh -c "exit 1" > manifests/fail-job.yaml
```

Thêm `backoffLimit: 2` vào `spec` trong `manifests/fail-job.yaml`:

```yaml
spec:
  backoffLimit: 2
  template:
    spec:
      containers:
        - name: fail-job
          image: busybox
          command: ["sh", "-c", "exit 1"]
      restartPolicy: Never
```

```bash
kubectl apply -f manifests/fail-job.yaml
kubectl get pods -l job-name=fail-job -w
kubectl describe job fail-job
```

**Vì sao thấy nhiều hơn 3 Pod đôi khi:** `backoffLimit: 2` nghĩa là Job được
phép **retry tối đa 2 lần** sau lần thử đầu tiên (tổng tối đa 3 lần thử),
với thời gian chờ tăng dần giữa các lần retry (exponential backoff — 10s,
20s, 40s...). Sau khi vượt quá, Job được đánh dấu điều kiện
`BackoffLimitExceeded` và ngừng tạo Pod mới, dù bản thân Job không tự xoá
các Pod đã fail (bạn vẫn thấy chúng ở trạng thái `Error` khi list).

Đây chính là `backoffLimit: 2` mà `sync-service` cũng dùng (xem Phần A) —
lý do thực tế đằng sau con số đó: một lần poll lỗi tạm thời (API chuyến bay
timeout, DB restart giữa chừng...) đáng để thử lại vài lần trước khi bỏ
cuộc, nhưng không nên retry vô hạn vì `concurrencyPolicy: Forbid` sẽ chặn
chu kỳ 5 phút kế tiếp cho tới khi lần chạy hiện tại kết thúc hẳn (thành
công hoặc hết `backoffLimit`).

## Phần C — CronJob của riêng bạn

```bash
kubectl create cronjob hello-cron \
  --image=busybox \
  --schedule="*/1 * * * *" \
  -- sh -c 'date; echo Hello from CKAD lab'

kubectl get cronjob hello-cron
kubectl get jobs --watch
# Ctrl+C sau khi thấy ít nhất 2 Job con (tên dạng hello-cron-<timestamp>)

kubectl logs job/<tên-job-con-vừa-thấy>
```

`kubectl get jobs`/`kubectl get cronjobs` ở bước này cũng liệt kê
`sync-service` bên cạnh `hello-cron` — phân biệt bằng tên. Cả Job tự sinh
theo lịch lẫn Job bạn tạo thủ công qua `--from=cronjob` (Phần A) đều mang
`ownerReferences` trỏ về CronJob — điểm khác nhau là `hello-cron`'s Job con
có thêm `blockOwnerDeletion: true`, còn `sync-manual-1` thì không, nhưng cả
hai đều bị cascade-xoá nếu bạn xoá CronJob tương ứng.

## Phần D — So sánh

| | Job | CronJob | Deployment |
|---|---|---|---|
| `restartPolicy` được phép trong Pod template | `Never` hoặc `OnFailure` (không được `Always`) | Giống Job — Pod template của CronJob **là** Pod template của Job nó sinh ra | Chỉ `Always` (mặc định, và cũng là giá trị duy nhất hợp lệ) |
| Owner chain của Pod | Job → Pod (`ownerReferences` trỏ thẳng vào Job) | CronJob → Job → Pod (2 tầng owner) — áp dụng cho **cả** Job tự sinh theo lịch lẫn Job tạo thủ công qua `--from=cronjob` (đã kiểm chứng ở Phần A) | Deployment → ReplicaSet → Pod (2 tầng owner) |
| Dùng khi nào | Tác vụ chạy-một-lần-rồi-xong: migration, batch tính toán, xử lý file một lần | Tác vụ lặp lại theo lịch cố định: backup định kỳ, dọn dẹp, báo cáo hàng đêm | Ứng dụng chạy liên tục, cần luôn có N bản sao sẵn sàng phục vụ traffic |

**Lý do `restartPolicy: Always` không hợp lệ cho Job/CronJob:** nếu container
luôn được restart bất kể exit code, khái niệm "hoàn thành" (`completions`)
không bao giờ đạt được — Job sẽ chạy vĩnh viễn, trái với bản chất
run-to-completion của nó. Đây là field kubectl sẽ từ chối ngay khi bạn
`apply` nếu set sai.

**Ví dụ thật trong repo này:** `sync-service` chính là một CronJob thay vì
Deployment — nó poll flight API một lần rồi thoát mỗi 5 phút, không phải
server chạy liên tục — đúng use case ở hàng "Dùng khi nào" của cột CronJob
phía trên. `bot-service`/`flight-service` (Lab 1.2) ngược lại là Deployment
vì chúng phải luôn sẵn sàng phục vụ request.

## Dọn dẹp

```bash
kubectl delete job fail-job sync-manual-1
kubectl delete cronjob hello-cron
```
