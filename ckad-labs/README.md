# CKAD Labs

Thư mục thực hành riêng cho kỳ thi **CKAD (Certified Kubernetes Application
Developer)**. Nội dung bài tập (`kubectl`/YAML) không liên quan gì về mặt
kiến trúc tới ứng dụng Flight Tracker — đây chỉ là project cho khoá học
CKAD, không phải hệ thống production, nên mọi lab chạy thẳng trong
namespace `flight-tracker` (namespace mà `deployments/helm/flight-tracker/deploy-local.sh`
/ `deploy-demo.sh` dùng để deploy app) mà không cần cô lập hay rào chắn gì
thêm.

## Chuẩn bị

- Một cluster để thực hành: `kind`, `minikube`, hoặc `k3d` đều được. Không
  cần cluster nhiều node.
- `kubectl` đã trỏ đúng context vào cluster đó (`kubectl config current-context`).
- Khuyến nghị (đúng những gì nên bật sẵn trước khi thi thật):
  ```bash
  alias k=kubectl
  export do="--dry-run=client -o yaml"
  export now="--force --grace-period=0"
  source <(kubectl completion bash)   # hoặc zsh/powershell tương ứng
  ```
- Một editor gõ nhanh trong terminal (vim/nano) — trong phòng thi không có
  chuột, không có VS Code. Nếu dùng vim, đáng để set sẵn:
  ```vim
  set nu ai et sw=2 ts=2
  ```

### Nếu bạn chạy trên Windows qua Git Bash

Git Bash tự động "dịch" bất kỳ tham số dòng lệnh nào trông giống đường dẫn
POSIX tuyệt đối (bắt đầu bằng `/`) sang đường dẫn Windows **trước khi**
truyền cho một binary native như `kubectl.exe` — kể cả khi tham số đó thực
ra là một đường dẫn *bên trong container* (chạy trên Linux), không phải
đường dẫn trên máy bạn. Ví dụ thật đã gặp khi viết lab này: `-- /bin/sh -c
'...'` bị biến thành `-- "C:/Program Files/Git/usr/bin/sh" -c '...'`, khiến
container crash-loop với lỗi `exec: ... no such file or directory`. Cách
né đơn giản nhất: dùng `sh` thay vì `/bin/sh` (PATH lookup vẫn tìm đúng)
bất cứ khi nào bạn tự gõ lệnh có đường dẫn tuyệt đối làm container command —
tất cả lệnh trong các lab này đã tránh sẵn kiểu này. Nếu bắt buộc phải dùng
đường dẫn tuyệt đối, có 2 cách khác: thêm `MSYS_NO_PATHCONV=1` trước lệnh,
hoặc gõ hai dấu `/` ở đầu (`//bin/sh`).

## Cách dùng mỗi lab

1. Mở `README.md` của lab, đọc mục tiêu + nhiệm vụ.
2. Tự làm trong đúng thời lượng ghi ở đầu bài (đây là luyện tốc độ, không
   phải luyện đáp án đúng).
3. Lưu các manifest bạn export/viết vào thư mục con `manifests/` của lab đó.
4. Sau khi xong (hoặc bí quá 2x thời gian quy định), mở `solution.md` để đối
   chiếu — mỗi solution có kèm giải thích *tại sao*, không chỉ lệnh.
5. Mỗi lab có `scripts/solve.sh` — bản thực thi được của toàn bộ
   `solution.md`, tự kiểm tra kết quả (`[PASS]`/`[FAIL]`) — và
   `scripts/cleanup.sh` để dọn dẹp. Phần nào dựa trên resource thật của
   project (`bot-service`, `sync-service`...) tự bỏ qua với cảnh báo nếu
   bạn chưa deploy app, thay vì làm script lỗi giữa chừng. Dùng để đối
   chiếu nhanh hoặc chạy lại môi trường sau khi luyện tay xong, không thay
   thế việc tự gõ lệnh.
6. Dọn dẹp trước khi qua lab tiếp theo (mỗi lab ghi rõ lệnh cleanup ở cuối,
   khớp với `scripts/cleanup.sh`).

## Domains

| Domain | Thư mục | Trạng thái |
|---|---|---|
| Application Design and Build (20%) | [01-application-design-and-build/](01-application-design-and-build/) | 4 lab |

Các domain khác (Configuration, Multi-Container Pods nâng cao, Observability,
Pod Design, Services & Networking, State Persistence) sẽ được thêm khi có
yêu cầu bài lab tương ứng.
