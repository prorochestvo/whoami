// progressive enhancement only — the page is fully readable without this file.
(() => {
  "use strict";

  // --- email click-to-copy ---
  // use event delegation so the handler survives locale swaps.
  document.addEventListener("click", (e) => {
    const a = e.target.closest('a[href^="mailto:"]');
    if (!a || !navigator.clipboard) return;
    const address = a.getAttribute("href").replace(/^mailto:/, "");
    e.preventDefault();
    navigator.clipboard.writeText(address).then(
      () => {
        const original = a.textContent;
        a.textContent = "copied ✓";
        setTimeout(() => { a.textContent = original; }, 1400);
      },
      () => { window.location.href = a.getAttribute("href"); }
    );
  });

  // --- locale fetch-and-swap ---

  const DEFAULT_LOCALE = "en";
  // cache of parsed locale JSON keyed by locale code
  const localeCache = new Map();
  // build stamp for cache-busting locale JSON fetches
  const stamp = document.querySelector('link[rel="stylesheet"]')
    ? new URL(document.querySelector('link[rel="stylesheet"]').href).searchParams.get("v") || ""
    : "";

  const progressBar = document.querySelector(".progress-bar");

  function showProgress() {
    if (!progressBar) return;
    document.body.classList.remove("lang-done");
    document.body.classList.add("lang-loading");
  }

  function hideProgress() {
    if (!progressBar) return;
    document.body.classList.remove("lang-loading");
    document.body.classList.add("lang-done");
    // remove lang-done after the CSS transition ends so the bar resets
    progressBar.addEventListener("transitionend", () => {
      document.body.classList.remove("lang-done");
    }, { once: true });
  }

  // fetchLocale fetches and caches the JSON for a locale code.
  function fetchLocale(code) {
    if (localeCache.has(code)) {
      return Promise.resolve(localeCache.get(code));
    }
    const url = "/content/" + code + ".json" + (stamp ? "?v=" + stamp : "");
    return fetch(url)
      .then((res) => {
        if (!res.ok) throw new Error("fetch " + code + " failed: " + res.status);
        return res.json();
      })
      .then((data) => {
        localeCache.set(code, data);
        return data;
      });
  }

  // localeURL returns the URL for a locale: default "en" -> "/", others -> "/<code>/"
  function localeURL(code) {
    return code === DEFAULT_LOCALE ? "/" : "/" + code + "/";
  }

  // swapPage applies the fetched locale data to the current DOM.
  // All values go through textContent / createElement — never innerHTML with
  // fetched data (XSS parity with html/template build-time auto-escaping).
  //
  // DOM sections this function mirrors from templates/index.html.tmpl:
  //   #whoami     — person name, title, tagline, .facts list (experience + contacts)
  //   #about      — .prose paragraphs
  //   #projects   — .project articles (name, repo/url, stack, summary, highlights)
  //   #experience — .job articles (role, company, period, location, arrangement,
  //                  summary, highlights, stack)
  //   #skills     — .skills dl / .skill-row dt+dd
  //   #education  — .edu articles + .langs-spoken span tags
  //   .lang-list  — locale switcher <a>/<span> elements
  //
  // If the template structure changes (section IDs, class names, nesting), this
  // function must be updated to match.
  function swapPage(data, code) {
    const p = data.person || {};

    // update <html lang>
    document.documentElement.lang = code;

    // update document.title
    const title = (p.name || "") + (p.title ? " — " + p.title : "");
    document.title = title;

    // update meta description
    const metaDesc = document.querySelector('meta[name="description"]');
    if (metaDesc) metaDesc.setAttribute("content", p.tagline || "");

    // update OG/Twitter title+description
    ["og:title", "twitter:title"].forEach((prop) => {
      const el = document.querySelector('meta[property="' + prop + '"], meta[name="' + prop + '"]');
      if (el) el.setAttribute("content", title);
    });
    ["og:description", "twitter:description"].forEach((prop) => {
      const el = document.querySelector('meta[property="' + prop + '"], meta[name="' + prop + '"]');
      if (el) el.setAttribute("content", p.tagline || "");
    });

    // update canonical (must be absolute) and og:url
    const abs = window.location.origin + localeURL(code);
    const canonical = document.querySelector('link[rel="canonical"]');
    if (canonical) canonical.setAttribute("href", abs);
    const ogURL = document.querySelector('meta[property="og:url"]');
    if (ogURL) ogURL.setAttribute("content", abs);

    // swap #whoami section
    const nameEl = document.getElementById("name");
    if (nameEl) setTextContent(nameEl, p.name || "");

    const roleEl = document.querySelector(".role");
    if (roleEl) setTextContent(roleEl, p.title || "");

    const taglineEl = document.querySelector(".tagline");
    if (taglineEl) setTextContent(taglineEl, p.tagline || "");

    // swap facts list (experience + contacts)
    const factsList = document.querySelector(".facts");
    if (factsList) {
      factsList.innerHTML = "";
      // experience fact
      const expLi = document.createElement("li");
      const expK = document.createElement("span");
      expK.className = "k";
      setTextContent(expK, "experience");
      const expV = document.createElement("span");
      expV.className = "v";
      setTextContent(expV, p.experience || "");
      expLi.appendChild(expK);
      expLi.appendChild(expV);
      factsList.appendChild(expLi);
      // contacts
      (data.contacts || []).forEach((c) => {
        const li = document.createElement("li");
        const k = document.createElement("span");
        k.className = "k";
        setTextContent(k, (c.label || "").toLowerCase());
        const v = document.createElement("span");
        v.className = "v";
        if (c.url && /^(https?|mailto|tel):/i.test(c.url)) {
          const a = document.createElement("a");
          a.href = c.url;
          setTextContent(a, c.value || "");
          v.appendChild(a);
        } else {
          setTextContent(v, c.value || "");
        }
        li.appendChild(k);
        li.appendChild(v);
        factsList.appendChild(li);
      });
    }

    // swap about section
    const aboutSection = document.getElementById("about");
    if (aboutSection) {
      const paras = aboutSection.querySelectorAll(".prose");
      paras.forEach((el) => el.remove());
      const aboutH = aboutSection.querySelector(".comment");
      (data.about || []).forEach((text) => {
        const p2 = document.createElement("p");
        p2.className = "prose";
        setTextContent(p2, text);
        aboutSection.appendChild(p2);
      });
      if (aboutH) aboutSection.insertBefore(aboutH, aboutSection.firstChild);
    }

    // swap projects section
    const projSection = document.getElementById("projects");
    if (projSection) {
      const projects = projSection.querySelectorAll(".project");
      projects.forEach((el) => el.remove());
      const projH = projSection.querySelector(".comment");
      (data.projects || []).forEach((proj) => {
        projSection.appendChild(buildProjectElement(proj));
      });
      if (projH) projSection.insertBefore(projH, projSection.firstChild);
    }

    // swap experience section
    const expSection = document.getElementById("experience");
    if (expSection) {
      const jobs = expSection.querySelectorAll(".job");
      jobs.forEach((el) => el.remove());
      const expH = expSection.querySelector(".comment");
      // sort in code (mirrors the Go build-time order): start (ISO YYYY-MM)
      // descending; ties by end descending where empty end = present (ranks
      // highest); remaining ties by company ascending. empty start sorts last.
      const experience = (data.experience || []).slice().sort((a, b) => {
        const sa = a.start || "", sb = b.start || "";
        if (sa !== sb) {
          if (sa === "") return 1;
          if (sb === "") return -1;
          return sa < sb ? 1 : -1;
        }
        const ea = a.end || "", eb = b.end || "";
        if (ea !== eb) {
          if (ea === "") return -1;
          if (eb === "") return 1;
          return ea < eb ? 1 : -1;
        }
        const ca = a.company || "", cb = b.company || "";
        return ca < cb ? -1 : (ca > cb ? 1 : 0);
      });
      experience.forEach((job) => {
        const article = buildJobElement(job);
        expSection.appendChild(article);
      });
      if (expH) expSection.insertBefore(expH, expSection.firstChild);
    }

    // swap skills section
    const skillsSection = document.getElementById("skills");
    if (skillsSection) {
      const dl = skillsSection.querySelector(".skills");
      if (dl) {
        dl.innerHTML = "";
        (data.skills || []).forEach((group) => {
          const div = document.createElement("div");
          div.className = "skill-row";
          const dt = document.createElement("dt");
          setTextContent(dt, group.name || "");
          const dd = document.createElement("dd");
          (group.items || []).forEach((item) => {
            const span = document.createElement("span");
            span.className = "tag";
            setTextContent(span, item);
            dd.appendChild(span);
          });
          div.appendChild(dt);
          div.appendChild(dd);
          dl.appendChild(div);
        });
      }
    }

    // swap education section
    const eduSection = document.getElementById("education");
    if (eduSection) {
      const eduArticles = eduSection.querySelectorAll(".edu");
      eduArticles.forEach((el) => el.remove());
      const eduH = eduSection.querySelector(".comment");
      const langsSpoken = eduSection.querySelector(".langs-spoken");
      (data.education || []).forEach((edu) => {
        const article = document.createElement("article");
        article.className = "edu";
        const h3 = document.createElement("h3");
        h3.className = "edu-inst";
        setTextContent(h3, edu.institution || "");
        const meta = document.createElement("p");
        meta.className = "edu-meta";
        setTextContent(meta, [edu.degree, edu.field, edu.year].filter(Boolean).join(" · "));
        article.appendChild(h3);
        article.appendChild(meta);
        if (langsSpoken) {
          eduSection.insertBefore(article, langsSpoken);
        } else {
          eduSection.appendChild(article);
        }
      });
      if (eduH) eduSection.insertBefore(eduH, eduSection.firstChild);
      // rebuild spoken languages
      if (langsSpoken) {
        langsSpoken.innerHTML = "";
        (data.languages || []).forEach((lang) => {
          const span = document.createElement("span");
          span.className = "tag";
          setTextContent(span, lang.name || "");
          langsSpoken.appendChild(span);
        });
      }
    }

    // update locale switcher: mark the current locale
    const links = document.querySelectorAll(".lang-list a.lang-link, .lang-list span.lang-current");
    links.forEach((el) => {
      const lc = el.dataset.locale || el.getAttribute("hreflang");
      if (!lc) return;
      if (lc === code) {
        // convert <a> to <span class="lang-current">; data-locale is required so
        // subsequent swaps can read the locale back from the non-link element.
        const span = document.createElement("span");
        span.className = "lang-v lang-current";
        span.setAttribute("aria-current", "page");
        span.dataset.locale = lc;
        setTextContent(span, lc);
        el.replaceWith(span);
      } else if (el.tagName === "SPAN") {
        // convert <span> back to <a>
        const a = document.createElement("a");
        a.className = "lang-v lang-link";
        a.href = localeURL(lc);
        a.setAttribute("hreflang", lc);
        a.dataset.locale = lc;
        setTextContent(a, lc);
        el.replaceWith(a);
      }
    });
  }

  // buildJobElement creates an <article class="job"> from job data.
  function buildJobElement(job) {
    const article = document.createElement("article");
    article.className = "job";

    const h3 = document.createElement("h3");
    h3.className = "job-role";
    setTextContent(h3, job.role || "");
    const at = document.createElement("span");
    at.className = "at";
    setTextContent(at, " @ " + (job.company || ""));
    h3.appendChild(at);
    if (job.current) {
      const now = document.createElement("span");
      now.className = "now";
      setTextContent(now, "active");
      h3.appendChild(document.createTextNode(" "));
      h3.appendChild(now);
    }

    const meta = document.createElement("p");
    meta.className = "job-meta";
    setTextContent(meta, [job.period, job.location, job.arrangement].filter(Boolean).join(" · "));

    const summary = document.createElement("p");
    summary.className = "job-summary";
    setTextContent(summary, job.summary || "");

    const ul = document.createElement("ul");
    ul.className = "job-highlights";
    (job.highlights || []).forEach((h) => {
      const li = document.createElement("li");
      setTextContent(li, h);
      ul.appendChild(li);
    });

    const stack = document.createElement("p");
    stack.className = "stack";
    (job.stack || []).forEach((item) => {
      const span = document.createElement("span");
      span.className = "tag";
      setTextContent(span, item);
      stack.appendChild(span);
    });

    article.appendChild(h3);
    article.appendChild(meta);
    article.appendChild(summary);
    article.appendChild(ul);
    article.appendChild(stack);
    return article;
  }

  // buildProjectElement creates an <article class="project"> from project data.
  // Mirrors the #projects markup in templates/index.html.tmpl.
  function buildProjectElement(proj) {
    const article = document.createElement("article");
    article.className = "project";

    const h3 = document.createElement("h3");
    h3.className = "project-name";
    const id = document.createElement("span");
    id.className = "project-id";
    setTextContent(id, proj.name || "");
    h3.appendChild(id);
    if (proj.repo) {
      const repo = document.createElement("span");
      repo.className = "project-repo";
      if (proj.url && /^(https?|mailto|tel):/i.test(proj.url)) {
        const a = document.createElement("a");
        a.href = proj.url;
        setTextContent(a, proj.repo);
        repo.appendChild(a);
      } else {
        setTextContent(repo, proj.repo);
      }
      h3.appendChild(repo);
    }
    article.appendChild(h3);

    if (proj.stack && proj.stack.length) {
      const stack = document.createElement("p");
      stack.className = "stack";
      proj.stack.forEach((item) => {
        const span = document.createElement("span");
        span.className = "tag";
        setTextContent(span, item);
        stack.appendChild(span);
      });
      article.appendChild(stack);
    }

    const summary = document.createElement("p");
    summary.className = "job-summary";
    setTextContent(summary, proj.summary || "");
    article.appendChild(summary);

    if (proj.highlights && proj.highlights.length) {
      const ul = document.createElement("ul");
      ul.className = "job-highlights";
      proj.highlights.forEach((h) => {
        const li = document.createElement("li");
        setTextContent(li, h);
        ul.appendChild(li);
      });
      article.appendChild(ul);
    }

    return article;
  }

  // setTextContent sets textContent on el — the only safe way to insert fetched
  // strings into the DOM (mirrors html/template auto-escaping at build time).
  function setTextContent(el, text) {
    el.textContent = text;
  }

  // intercept locale link clicks to swap content without a full reload
  document.addEventListener("click", (e) => {
    const a = e.target.closest("a.lang-link[data-locale]");
    if (!a) return;
    e.preventDefault();
    const code = a.dataset.locale;
    const href = a.href;

    showProgress();
    fetchLocale(code)
      .then((data) => {
        swapPage(data, code);
        history.pushState({ locale: code }, "", localeURL(code));
        hideProgress();
      })
      .catch(() => {
        // fetch failed — fall back to normal navigation
        hideProgress();
        window.location.href = href;
      });
  });

  // handle back/forward navigation
  window.addEventListener("popstate", (e) => {
    const code = (e.state && e.state.locale) ? e.state.locale : DEFAULT_LOCALE;
    fetchLocale(code)
      .then((data) => { swapPage(data, code); })
      .catch(() => { window.location.reload(); });
  });

  // seed current locale in history state so popstate always has a locale to read
  const currentLang = document.documentElement.lang || DEFAULT_LOCALE;
  history.replaceState({ locale: currentLang }, "", window.location.href);

})();
