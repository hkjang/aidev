/**
 * Invenqor Landing Page Interactive Logic
 * Features: Dynamic Language Switcher (KR/EN), Showcase Tabs, Clipboard Copy, FAQ Accordion
 */

document.addEventListener('DOMContentLoaded', () => {
  // 1. Language Toggle Logic
  const langBtns = document.querySelectorAll('.lang-btn');
  const i18nElements = document.querySelectorAll('[data-i18n-kr]');
  
  function setLanguage(lang) {
    // Update active button state
    langBtns.forEach(btn => {
      if (btn.dataset.lang === lang) {
        btn.classList.add('active');
      } else {
        btn.classList.remove('active');
      }
    });

    // Update document lang tag
    document.documentElement.lang = lang === 'en' ? 'en' : 'ko';

    // Update text contents
    i18nElements.forEach(el => {
      const text = el.getAttribute(`data-i18n-${lang}`);
      if (text) {
        if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
          el.placeholder = text;
        } else {
          el.innerHTML = text;
        }
      }
    });

    // Save choice
    localStorage.setItem('invenqor_lang', lang);
  }

  langBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const selectedLang = btn.dataset.lang;
      setLanguage(selectedLang);
    });
  });

  // Load saved language or default to 'kr'
  const savedLang = localStorage.getItem('invenqor_lang') || 'kr';
  setLanguage(savedLang);

  // 2. Showcase Window Tabs
  const tabBtns = document.querySelectorAll('.tab-btn');
  const tabContents = document.querySelectorAll('.tab-content');

  tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const targetTab = btn.dataset.tab;

      tabBtns.forEach(b => b.classList.remove('active'));
      tabContents.forEach(c => c.classList.remove('active'));

      btn.classList.add('active');
      document.getElementById(targetTab)?.classList.add('active');
    });
  });

  // 3. FAQ Accordions
  const faqItems = document.querySelectorAll('.faq-item');

  faqItems.forEach(item => {
    const questionBtn = item.querySelector('.faq-question');
    questionBtn?.addEventListener('click', () => {
      const isActive = item.classList.contains('active');

      // Close all other items for clean accordion UX
      faqItems.forEach(i => i.classList.remove('active'));

      if (!isActive) {
        item.classList.add('active');
      }
    });
  });

  // 4. Code Snippet Copy Buttons
  const copyBtns = document.querySelectorAll('.btn-copy');

  copyBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const targetId = btn.dataset.copyTarget;
      const targetEl = document.getElementById(targetId);

      if (targetEl) {
        const textToCopy = targetEl.innerText || targetEl.textContent;
        navigator.clipboard.writeText(textToCopy).then(() => {
          const currentLang = localStorage.getItem('invenqor_lang') || 'kr';
          const originalText = btn.innerText;
          btn.innerText = currentLang === 'en' ? 'Copied!' : '복사완료!';
          btn.style.background = 'rgba(6, 182, 212, 0.3)';
          btn.style.color = '#fff';

          setTimeout(() => {
            btn.innerText = originalText;
            btn.style.background = '';
            btn.style.color = '';
          }, 2000);
        }).catch(err => {
          console.error('Copy failed:', err);
        });
      }
    });
  });
});


  // Mobile Menu Toggle
  const mobileBtn = document.getElementById('mobile-menu-btn');
  const navLinks = document.querySelector('.nav-links') || document.querySelector('.nav-menu') || document.querySelector('.nav-list');
  if (mobileBtn && navLinks) {
    mobileBtn.addEventListener('click', () => {
      navLinks.classList.toggle('active');
    });
    navLinks.querySelectorAll('a').forEach(link => {
      link.addEventListener('click', () => navLinks.classList.remove('active'));
    });
  }
