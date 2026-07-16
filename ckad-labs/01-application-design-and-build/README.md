# Domain 1 — Application Design and Build (20%)

Domain này trong CKAD curriculum bao trùm: chọn đúng loại workload resource
(Pod / Deployment / Job / CronJob) cho từng bài toán, thiết kế multi-container
Pod (init container, sidecar), và quản lý object bằng label/annotation/selector.
Cả 4 kỹ năng đều là nền tảng — hầu hết các domain khác trong đề thi build
thêm lên trên chúng.

## Labs

| Lab | Thời lượng | Nội dung |
|---|---|---|
| [1.1 — The 60-Second Pod](lab-1.1-the-60-second-pod/) | ~45 phút | Tạo Pod imperatively (labels, env, resources), export YAML, verify không dùng editor |
| [1.2 — Init + Sidecar Pattern](lab-1.2-init-sidecar-pattern/) | ~60 phút | Đọc hiểu sidecar thật của `bot-service` (2 container chia sẻ `emptyDir`) + tự viết Pod init container/app/sidecar |
| [1.3 — Jobs & CronJobs](lab-1.3-jobs-cronjobs/) | ~45 phút | Đọc/trigger thủ công CronJob thật (`sync-service`), Job cố tình fail quan sát `backoffLimit`, tự tạo CronJob, phân biệt với Deployment |
| [1.4 — Label & Annotation Drill](lab-1.4-label-annotation-drill/) | ~30 phút | Tạo hàng loạt Pod, cập nhật label bằng selector, `--overwrite`, cộng thêm truy vấn/label/annotate resource thật (`bot-service`) để phân biệt sửa object đang chạy vs sửa desired state |

Tổng: ~3 giờ. Làm theo đúng thứ tự — lab 1.2 và 1.3 giả định bạn đã quen thao
tác export/apply YAML từ lab 1.1.
