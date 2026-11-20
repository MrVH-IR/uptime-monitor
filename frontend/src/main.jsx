import React, { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { getMonitors, getResults, getLatestResults } from "./api";
import "./style.css";
const date = (value) => (value ? new Date(value).toLocaleString("fa-IR") : "—");
const state = (result) =>
  !result ? "pending" : result.IsSuccess ? "up" : "down";
const labels = { up: "در دسترس", down: "اختلال", pending: "در انتظار بررسی" };
function Pager({ page, more, busy, change }) {
  return (
    <div className="pager">
      <button disabled={page === 1 || busy} onClick={() => change(page - 1)}>
        صفحه قبل
      </button>
      <span>صفحه {page}</span>
      <button disabled={!more || busy} onClick={() => change(page + 1)}>
        صفحه بعد
      </button>
    </div>
  );
}
function App() {
  const [page, setPage] = useState(1),
    [more, setMore] = useState(false),
    [historyPage, setHistoryPage] = useState(1),
    [historyMore, setHistoryMore] = useState(false);
  const [monitors, setMonitors] = useState([]),
    [latest, setLatest] = useState([]),
    [selected, setSelected] = useState(null),
    [history, setHistory] = useState([]);
  const [error, setError] = useState(""),
    [historyError, setHistoryError] = useState(""),
    [loading, setLoading] = useState(true),
    [historyLoading, setHistoryLoading] = useState(false),
    [updated, setUpdated] = useState(null),
    [refresh, setRefresh] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let timer;
    async function load() {
      try {
        const data = await getMonitors(page, controller.signal);
        const m = data.items;
        const results = m.length
          ? await getLatestResults(
              m.map((item) => item.ID),
              controller.signal,
            )
          : { items: [] };
        setMonitors(m);
        setMore(data.hasMore);
        setLatest(results.items);
        setError("");
        setUpdated(new Date());
        setSelected((old) =>
          m.some((x) => x.ID === old) ? old : (m[0]?.ID ?? null),
        );
      } catch (e) {
        if (e.name !== "AbortError") setError(e.message);
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
          timer = setTimeout(load, 10000);
        }
      }
    }
    load();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [refresh, page]);
  useEffect(() => {
    if (!selected) return;
    const controller = new AbortController();
    let timer;
    setHistory([]);
    setHistoryLoading(true);
    setHistoryError("");
    async function load() {
      try {
        const data = await getResults(selected, historyPage, controller.signal);
        setHistory(data.items);
        setHistoryMore(data.hasMore);
        setHistoryError("");
      } catch (e) {
        if (e.name !== "AbortError") setHistoryError(e.message);
      } finally {
        if (!controller.signal.aborted) {
          setHistoryLoading(false);
          timer = setTimeout(load, 10000);
        }
      }
    }
    load();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [selected, refresh, historyPage]);
  const active = monitors.find((m) => m.ID === selected),
    resultFor = (id) => latest.find((r) => r.MonitorID === id),
    up = monitors.filter((m) => resultFor(m.ID)?.IsSuccess).length,
    down = monitors.filter((m) => state(resultFor(m.ID)) === "down").length;
  return (
    <div className="shell">
      <aside>
        <a className="brand" href="/">
          ◉ <strong>دیدبان</strong>
        </a>
        <span className="subtitle">UPTIME MONITOR</span>
        <div className="nav">▦ &nbsp; نمای کلی سرویس‌ها</div>
        <div className="aside-bottom">
          پایش ساده، دید روشن
          <br />
          <small>تازه‌سازی خودکار هر ۱۰ ثانیه</small>
        </div>
      </aside>
      <main>
        <header>
          <div>
            <span className="eyebrow">مرکز پایش</span>
            <h1>نبض سرویس‌های شما</h1>
            <p>وضعیت دسترسی و عملکرد، در یک نگاه.</p>
          </div>
          <button onClick={() => setRefresh((x) => x + 1)}>
            ↻ &nbsp; به‌روزرسانی
          </button>
        </header>
        {error && (
          <div role="alert" className="error">
            {error}
          </div>
        )}
        <section className="stats" aria-label="خلاصه وضعیت">
          <article>
            <span>مانیتورهای این صفحه</span>
            <b>{monitors.length}</b>
            <small>سرویس ثبت‌شده</small>
          </article>
          <article>
            <span>در دسترس</span>
            <b className="green">{up}</b>
            <small>آخرین بررسی موفق</small>
          </article>
          <article>
            <span>دارای اختلال</span>
            <b className="red">{down}</b>
            <small>نیازمند بررسی</small>
          </article>
          <article>
            <span>در انتظار بررسی</span>
            <b>{monitors.length - up - down}</b>
            <small>بدون نتیجه ثبت‌شده</small>
          </article>
        </section>
        <section className="panel">
          <div className="section-head">
            <h2>سرویس‌ها</h2>
            <small>
              {updated
                ? `آخرین دریافت: ${updated.toLocaleTimeString("fa-IR")}`
                : "در حال دریافت…"}
            </small>
          </div>
          {loading ? (
            <p className="empty" role="status">
              در حال دریافت مانیتورها…
            </p>
          ) : monitors.length === 0 ? (
            <p className="empty">هنوز مانیتوری ثبت نشده است.</p>
          ) : (
            <div className="monitor-grid">
              {monitors.map((m) => {
                const r = resultFor(m.ID),
                  s = state(r);
                return (
                  <button
                    key={m.ID}
                    className={`monitor ${selected === m.ID ? "selected" : ""}`}
                    onClick={() => {
                      setSelected(m.ID);
                      setHistoryPage(1);
                    }}
                    aria-pressed={selected === m.ID}
                  >
                    <div className="card-top">
                      <strong>{m.Name}</strong>
                      <span className={`badge ${s}`}>{labels[s]}</span>
                    </div>
                    <div className="url" dir="ltr">
                      {m.URL}
                    </div>
                    <div className="card-bottom">
                      <span>
                        زمان پاسخ{" "}
                        <b dir="ltr">
                          {r?.ResponseTimeMS != null
                            ? `${r.ResponseTimeMS} ms`
                            : "—"}
                        </b>
                      </span>
                      <span>هر {m.IntervalSeconds} ثانیه</span>
                    </div>
                    <small>
                      {m.IsActive ? "فعال" : "متوقف"} · {m.HTTPMethod}
                    </small>
                  </button>
                );
              })}
            </div>
          )}
          <Pager
            page={page}
            more={more}
            busy={loading}
            change={(value) => {
              setLoading(true);
              setHistoryPage(1);
              setPage(value);
            }}
          />
        </section>
        {active && (
          <section className="panel">
            <div className="section-head">
              <div>
                <h2>تاریخچه بررسی‌ها</h2>
                <p dir="ltr">{active.Name}</p>
              </div>
              <small>۵۰ نتیجه در هر صفحه · جدیدترین در ابتدا</small>
            </div>
            {historyError && (
              <p className="error" role="alert">
                {historyError}
              </p>
            )}
            {historyLoading ? (
              <p className="empty" role="status">
                در حال دریافت تاریخچه…
              </p>
            ) : history.length === 0 ? (
              <p className="empty">اولین نتیجه هنوز ثبت نشده است.</p>
            ) : (
              <>
                <div className="timeline" aria-label="نتایج اخیر">
                  {history
                    .slice(0, 60)
                    .reverse()
                    .map((r) => (
                      <span
                        key={r.ID}
                        className={state(r)}
                        title={`${date(r.StartedAt)} · ${labels[state(r)]}`}
                      />
                    ))}
                </div>
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>زمان بررسی</th>
                        <th>وضعیت</th>
                        <th>کد HTTP</th>
                        <th>زمان پاسخ</th>
                        <th>جزئیات خطا</th>
                      </tr>
                    </thead>
                    <tbody>
                      {history.map((r) => (
                        <tr key={r.ID}>
                          <td>{date(r.StartedAt)}</td>
                          <td>
                            <span className={`badge ${state(r)}`}>
                              {labels[state(r)]}
                            </span>
                          </td>
                          <td>{r.StatusCode ?? "—"}</td>
                          <td dir="ltr">{r.ResponseTimeMS ?? "—"} ms</td>
                          <td className="error-detail" dir="ltr">
                            {r.ErrorMessage || "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
            <Pager
              page={historyPage}
              more={historyMore}
              busy={historyLoading}
              change={(value) => {
                setHistoryLoading(true);
                setHistoryPage(value);
              }}
            />
          </section>
        )}
        <footer>دیدبان · داده‌های واقعی از PostgreSQL</footer>
      </main>
    </div>
  );
}
createRoot(document.getElementById("root")).render(<App />);
