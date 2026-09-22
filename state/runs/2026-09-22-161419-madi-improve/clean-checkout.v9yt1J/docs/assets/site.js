const toggle = document.querySelector(".mobile-toggle");
const navigation = document.querySelector(".nav");
toggle?.addEventListener("click", () => {
  const open = toggle.getAttribute("aria-expanded") !== "true";
  toggle.setAttribute("aria-expanded", String(open));
  toggle.setAttribute("aria-label", open ? "탐색 메뉴 닫기" : "탐색 메뉴 열기");
  navigation?.classList.toggle("is-open", open);
});
navigation?.querySelectorAll("a").forEach((link) =>
  link.addEventListener("click", () => {
    toggle?.setAttribute("aria-expanded", "false");
    navigation.classList.remove("is-open");
  }),
);

const dialog = document.querySelector(".lightbox");
if (dialog) {
  const preview = dialog.querySelector("img");
  const caption = dialog.querySelector("[data-caption]");
  document.querySelectorAll("[data-screenshot]").forEach((button) => {
    button.addEventListener("click", () => {
      const source = button.querySelector("img");
      if (!source || !preview) return;
      preview.src = source.src;
      preview.alt = source.alt;
      caption.textContent = source.alt;
      dialog.showModal();
    });
  });
  dialog
    .querySelector(".lightbox-close")
    ?.addEventListener("click", () => dialog.close());
  dialog.addEventListener("click", (event) => {
    if (event.target === dialog) {
      const bounds = dialog.getBoundingClientRect();
      if (
        event.clientX < bounds.left ||
        event.clientX > bounds.right ||
        event.clientY < bounds.top ||
        event.clientY > bounds.bottom
      )
        dialog.close();
    }
  });
}

const galleryQuery = document.querySelector("#gallery-query");
const galleryCategory = document.querySelector("#gallery-category");
if (galleryQuery && galleryCategory) {
  const params = new URLSearchParams(location.search);
  galleryQuery.value = params.get("q") || "";
  const requestedCategory = params.get("category") || "";
  if (
    [...galleryCategory.options].some(
      (option) => option.value === requestedCategory,
    )
  )
    galleryCategory.value = requestedCategory;
  const filter = () => {
    const query = galleryQuery.value.trim().toLocaleLowerCase("ko"),
      category = galleryCategory.value;
    let visible = 0;
    document.querySelectorAll("[data-gallery-card]").forEach((card) => {
      card.hidden = Boolean(
        (category && card.dataset.category !== category) ||
        !card.dataset.title.toLocaleLowerCase("ko").includes(query),
      );
      if (!card.hidden) visible++;
    });
    document.querySelectorAll("[data-gallery-group]").forEach((group) => {
      group.hidden = !group.querySelector("[data-gallery-card]:not([hidden])");
    });
    document.querySelector("#gallery-count").textContent = visible
      ? `${visible}개 화면`
      : "조건에 맞는 화면이 없습니다. 검색어 또는 분류를 바꿔주세요.";
    const next = new URL(location.href);
    if (query) next.searchParams.set("q", galleryQuery.value.trim());
    else next.searchParams.delete("q");
    if (category) next.searchParams.set("category", category);
    else next.searchParams.delete("category");
    history.replaceState(null, "", next);
  };
  galleryQuery.form.addEventListener("submit", (event) =>
    event.preventDefault(),
  );
  galleryQuery.addEventListener("input", filter);
  galleryCategory.addEventListener("change", filter);
  filter();
}
