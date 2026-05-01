"use client";

import { useEffect, useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import { EXAMPLES, type Example } from "./examples";
import styles from "./page.module.css";

type EvaluateResponse = {
  result?: unknown;
  error?: string;
  parsedInput?: {
    method: string;
    url: string;
    header: Record<string, string[]>;
    body: unknown;
    rawBody: string;
    bodyParsed: boolean;
  };
};

const DEBOUNCE_MS = 300;

function useDarkMode() {
  const [dark, setDark] = useState(false);
  useEffect(() => {
    const m = window.matchMedia("(prefers-color-scheme: dark)");
    setDark(m.matches);
    const onChange = (e: MediaQueryListEvent) => setDark(e.matches);
    m.addEventListener("change", onChange);
    return () => m.removeEventListener("change", onChange);
  }, []);
  return dark;
}

export default function Home() {
  const [celExpression, setCelExpression] = useState<string>(
    EXAMPLES[0].celExpression,
  );
  const [httpRequest, setHttpRequest] = useState<string>(
    EXAMPLES[0].httpRequest,
  );
  const [response, setResponse] = useState<EvaluateResponse | null>(null);
  const [evaluating, setEvaluating] = useState(false);
  const [activeExample, setActiveExample] = useState<string>(EXAMPLES[0].name);
  const dark = useDarkMode();
  const reqIdRef = useRef(0);

  const loadExample = (ex: Example) => {
    setCelExpression(ex.celExpression);
    setHttpRequest(ex.httpRequest);
    setActiveExample(ex.name);
  };

  useEffect(() => {
    if (!celExpression.trim() || !httpRequest.trim()) {
      setResponse(null);
      return;
    }
    const handle = setTimeout(async () => {
      const myId = ++reqIdRef.current;
      setEvaluating(true);
      try {
        const res = await fetch("/api/evaluate", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ celExpression, httpRequest }),
        });
        const data: EvaluateResponse = await res.json();
        if (myId === reqIdRef.current) setResponse(data);
      } catch (err) {
        if (myId === reqIdRef.current) {
          const msg = err instanceof Error ? err.message : String(err);
          setResponse({ error: msg });
        }
      } finally {
        if (myId === reqIdRef.current) setEvaluating(false);
      }
    }, DEBOUNCE_MS);
    return () => clearTimeout(handle);
  }, [celExpression, httpRequest]);

  const badgeClass = response?.error
    ? styles.statusBadgeError
    : response?.result === true
      ? styles.statusBadgeTrue
      : response?.result === false
        ? styles.statusBadgeFalse
        : styles.statusBadgeNeutral;

  const editorOptions = {
    minimap: { enabled: false },
    fontSize: 13,
    wordWrap: "on" as const,
    scrollBeyondLastLine: false,
    padding: { top: 8, bottom: 8 },
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1>Tekton CEL Playground</h1>
        <p>
          Evaluate{" "}
          <a
            href="https://github.com/google/cel-spec"
            target="_blank"
            rel="noreferrer"
          >
            CEL
          </a>{" "}
          expressions against a raw HTTP request, the way{" "}
          <a
            href="https://tekton.dev/docs/triggers/cel_expressions/"
            target="_blank"
            rel="noreferrer"
          >
            Tekton Triggers
          </a>{" "}
          does. Bindings: <code>body</code>, <code>header</code>,{" "}
          <code>requestURL</code>, <code>method</code>. Helpers:{" "}
          <code>header.match(key, value)</code>,{" "}
          <code>header.canonical(key)</code>.
        </p>
      </header>

      <section className={styles.section}>
        <div className={styles.label}>Examples</div>
        <div className={styles.examples}>
          {EXAMPLES.map((ex) => (
            <button
              key={ex.name}
              onClick={() => loadExample(ex)}
              title={ex.description}
              className={`${styles.exampleBtn} ${activeExample === ex.name ? styles.exampleBtnActive : ""}`}
            >
              {ex.name}
            </button>
          ))}
        </div>
      </section>

      <div className={styles.editorRow}>
        <div className={styles.editorCol}>
          <div className={styles.editorTitle}>
            <span>CEL Expression</span>
          </div>
          <div className={styles.editorBox}>
            <Editor
              height="320px"
              defaultLanguage="javascript"
              theme={dark ? "vs-dark" : "light"}
              value={celExpression}
              onChange={(v) => setCelExpression(v ?? "")}
              options={editorOptions}
            />
          </div>
        </div>
        <div className={styles.editorCol}>
          <div className={styles.editorTitle}>
            <span>HTTP Request</span>
            <span className={styles.editorHint}>
              Content-Length is auto-fixed
            </span>
          </div>
          <div className={styles.editorBox}>
            <Editor
              height="320px"
              defaultLanguage="plaintext"
              theme={dark ? "vs-dark" : "light"}
              value={httpRequest}
              onChange={(v) => setHttpRequest(v ?? "")}
              options={editorOptions}
            />
          </div>
        </div>
      </div>

      <div className={styles.statusRow}>
        {response && (
          <span className={`${styles.statusBadge} ${badgeClass}`}>
            {response.error
              ? "error"
              : `result: ${JSON.stringify(response.result)}`}
          </span>
        )}
        {evaluating && (
          <span className={styles.statusEvaluating}>evaluating…</span>
        )}
      </div>

      {response && (
        <div className={styles.results}>
          <div className={styles.resultCol}>
            <div className={styles.label}>
              {response.error ? "Error" : "Result"}
            </div>
            <pre
              className={`${styles.resultPre} ${response.error ? styles.resultPreError : ""}`}
            >
              {response.error ?? JSON.stringify(response.result, null, 2)}
            </pre>
          </div>
          {response.parsedInput && (
            <div className={styles.resultCol}>
              <div className={styles.label}>Parsed Input</div>
              <pre className={styles.resultPre}>
                {JSON.stringify(response.parsedInput, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
