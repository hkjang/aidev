import { useId, useState } from "react";
import { useApp } from "../context";
import { Button } from "../ui";
import { suggestDate } from "./naturalDate";
import "./mobile.css";
export default function NaturalDateInput({
  label,
  value,
  onChange,
  disabled = false,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  const id = useId(),
    { user } = useApp(),
    [text, setText] = useState("");
  const timezone = user.preferences.timezone || "Asia/Seoul";
  const suggestion = suggestDate(text, timezone);
  return (
    <div className="field natural-date-field">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        type="date"
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
      />
      <details>
        <summary>말로 날짜 찾기</summary>
        <label htmlFor={`${id}-suggest`}>{label} 날짜 표현</label>
        <input
          id={`${id}-suggest`}
          value={text}
          disabled={disabled}
          maxLength={80}
          placeholder="내일, 다음주 월요일, 9월 20일"
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") e.preventDefault();
          }}
        />
        {text && (
          <div aria-live="polite">
            {suggestion ? (
              <>
                <p>
                  <strong>{suggestion.date}</strong>
                  <br />
                  {suggestion.explanation}
                  <br />
                  기준 시간대: {suggestion.timezone}
                </p>
                <Button
                  type="button"
                  disabled={disabled}
                  onClick={() => {
                    const current = suggestDate(text, timezone);
                    if (current?.date === suggestion.date) {
                      onChange(current.date);
                      setText("");
                    }
                  }}
                >
                  확인한 날짜 입력
                </Button>
                <small>
                  입력칸만 바뀝니다. 실제 저장은 기존 저장 버튼으로 확인하세요.
                </small>
              </>
            ) : (
              <p>날짜를 확정할 수 없습니다. 연도·월·일을 직접 선택해 주세요.</p>
            )}
          </div>
        )}
      </details>
    </div>
  );
}
