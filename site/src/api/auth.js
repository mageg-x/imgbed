import request from "./request";

export const authApi = {
  login(password) {
    return request.post("/auth/login", { password });
  },
  adminLogin(username, password) {
    return request.post("/auth/admin/login", { username, password });
  },
  logout() {
    return request.post("/auth/logout");
  },
  check() {
    return request.get("/auth/check");
  },
  refresh() {
    return request.post("/auth/refresh");
  },
};
