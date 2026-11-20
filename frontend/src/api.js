async function get(path, signal) {
  const response = await fetch(`/api/${path}`, { signal });
  if (!response.ok)
    throw new Error("دریافت اطلاعات ناموفق بود. اتصال به سرویس را بررسی کنید.");
  return {
    items: await response.json(),
    hasMore: response.headers.get("X-Has-More") === "true",
  };
}
export const getMonitors = (page, signal) =>
  get(`monitors?page=${page}`, signal);
export const getResults = (id, page, signal) =>
  get(`results?page=${page}${id ? `&monitor_id=${id}` : ""}`, signal);

export const getLatestResults = (ids, signal) =>
  get(`results?monitor_ids=${ids.join(",")}`, signal);
