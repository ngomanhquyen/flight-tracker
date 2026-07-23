# Domain 3 — Application Environment, Configuration and Security (25%)

Domain này trong CKAD curriculum bao trùm: cấu hình app qua ConfigMap/Secret
(cả 2 cách tiêu thụ — env var và volume mount), khoá chặt Pod bằng
SecurityContext (non-root, read-only rootfs, drop capabilities), phân quyền
truy cập Kubernetes API qua ServiceAccount + RBAC, và giới hạn tài nguyên
cấp namespace bằng ResourceQuota/LimitRange. Đây là domain có tỷ trọng cao
nhất trong đề thi (25%).

## Labs

| Lab | Thời lượng | Nội dung |
|---|---|---|
| [3.1 — ConfigMap & Secret Injection](lab-3.1-configmap-secret-injection/) | ~45 phút | Tạo Secret từ file + ConfigMap từ literal, inject cả 2 kiểu (env var + volume mount) trong 1 Pod; đối chiếu với pattern `envFrom` mà mọi service thật trong project đang dùng |
| [3.2 — Security Context Lockdown](lab-3.2-security-context-lockdown/) | ~45 phút | Non-root, read-only rootfs, drop toàn bộ capabilities, tắt privilege escalation — đối chiếu với việc `_helpers.tpl` hiện chưa set `securityContext` nào dù image các service Go đã dùng base distroless nonroot |
| [3.3 — ServiceAccount & RBAC](lab-3.3-serviceaccount-rbac/) | ~60 phút | Tạo ServiceAccount + Role + RoleBinding, Pod dùng token gọi thẳng Kubernetes API để list Pod thật trong `flight-tracker` — không service nào trong project hiện gọi K8s API, đây là bài tập tự chứa hoàn toàn |
| [3.4 — Namespace Quotas](lab-3.4-namespace-quotas/) | ~45 phút | Áp `ResourceQuota`/`LimitRange` lên namespace `flight-tracker` đang chạy thật, quan sát Pod bị `Pending` khi vượt quota — nối tiếp trực tiếp bug thiếu `resources.requests` đã phát hiện và fix ở lab 2.3 |

Tổng: ~3 giờ 15 phút.

**Trạng thái hiện tại:** mỗi lab mới có `README.md` (mô tả nhiệm vụ + cách áp
dụng vào project) — chưa có `solution.md` (lệnh cụ thể đã verify) hay
`scripts/solve.sh`/`cleanup.sh`. Sẽ bổ sung dần từng lab khi thực hành thật,
đúng cách domain 2 đã làm (viết `solution.md` sau khi thực hành, không phải
trước).

Lab 3.2 có 1 quyết định còn để ngỏ, xem mục "Thảo luận" trong
`lab-3.2-security-context-lockdown/README.md`: có nên áp SecurityContext
thẳng vào `common.deployment`/`common.cronjob` (cải tiến thật cho project)
hay chỉ luyện trên Pod độc lập.
