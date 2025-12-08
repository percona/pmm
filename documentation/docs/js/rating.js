function initFeedbackWidget() {
  const widget = document.querySelector(".rating-stars");
  if (!widget || widget.dataset.initialized === "true") return; // ✅ already initialized
  widget.dataset.initialized = "true";

  const stars = widget.querySelectorAll(".star");
  const feedbackForm = document.getElementById("feedback-form");
  const feedbackTextarea = document.getElementById("feedback-text");
  const emailInput = document.getElementById("feedback-email");
  const submitButton = document.getElementById("submit-feedback");
  const statusDiv = document.getElementById("feedback-status");
  const closeButton = document.getElementById("close-feedback");
  const notification = document.getElementById("feedback-notification");
  const submitNotification = document.getElementById("feedback-submit-notification");

  if (!stars.length || !feedbackForm) return;

  const formURL = "https://docs.google.com/forms/d/e/1FAIpQLSdXWLhl1oKQqq9AgG_Ugxs5EbF3NMAiddWswL4ITyaQZZr5BA/formResponse";
  const fieldIds = {
    rating: "entry.303027158",
    pageUrl: "entry.295622433",
    feedback: "entry.2131350495",
    email: "entry.1689727303"
  };

  let selectedRating = 0;
  let notificationTimeout;
  let ratingLocked = false; // 🔒 cooldown lock

  function showNotification(msg) {
    notification.textContent = msg;
    notification.style.display = "block";
    clearTimeout(notificationTimeout);
    notificationTimeout = setTimeout(() => {
      notification.style.display = "none";
    }, 3000);
  }

  function sendRatingOnly(rating) {
    const formData = new FormData();
    formData.append(fieldIds.rating, rating);
    formData.append(fieldIds.pageUrl, window.location.href);

    fetch(formURL, {
      method: "POST",
      mode: "no-cors",
      body: formData
    }).catch(() => {
      console.warn("Rating failed to submit silently.");
    });
  }

  function submitFeedback(rating, feedbackText = "", email = "") {
    const formData = new FormData();
    formData.append(fieldIds.rating, rating);
    formData.append(fieldIds.pageUrl, window.location.href);
    formData.append(fieldIds.feedback, feedbackText);
    if (email) formData.append(fieldIds.email, email);

    fetch(formURL, {
      method: "POST",
      mode: "no-cors",
      body: formData
    }).then(() => {
      statusDiv.style.color = "";
      feedbackTextarea.value = "";
      emailInput.value = "";
      feedbackForm.style.display = "none";
      selectedRating = 0;

      stars.forEach((s) => {
        s.textContent = "☆";
        s.classList.remove("active");
        s.classList.remove("hovered");
      });

      submitNotification.style.display = "block";
      clearTimeout(notificationTimeout);
      notificationTimeout = setTimeout(() => {
        submitNotification.style.display = "none";
      }, 4000);
    }).catch(() => {
      statusDiv.style.color = "";
      statusDiv.textContent = "Error sending feedback.";
    });
  }

  stars.forEach((star) => {
    const rating = parseInt(star.dataset.rating, 10);

    star.addEventListener("mouseover", () => {
      if (ratingLocked) return;
      stars.forEach((s, index) => {
        s.textContent = index < rating ? "★" : "☆";
        if(index < rating) {
          s.classList.add("hovered");
          s.classList.remove("active");
        } else {
          s.classList.remove("hovered");
          s.classList.remove("active");
        }
      });
    });

    star.addEventListener("mouseleave", () => {
      if (ratingLocked) return;
      stars.forEach((s, index) => {
        s.textContent = index < selectedRating ? "★" : "☆";
        if(index < selectedRating) {
          s.classList.add("active");
        } else {
          s.classList.remove("active");
        }
        s.classList.remove("hovered");
      });
    });

    star.addEventListener("click", () => {
      if (ratingLocked) return;
      ratingLocked = true; // 🔒 lock stars for 10 seconds

      selectedRating = rating;

      stars.forEach((s, index) => {
        s.textContent = index < rating ? "★" : "☆";
        if(index < rating) {
          s.classList.add("active");
        } else {
          s.classList.remove("active");
        }
        s.classList.remove("hovered");
      });

      sendRatingOnly(rating);
      feedbackForm.style.display = "block";
      statusDiv.textContent = "";

      setTimeout(() => {
        ratingLocked = false; // ⏱ unlock after 10 seconds
      }, 10000);
    });
  });

  submitButton.addEventListener("click", () => {
    if (!selectedRating) return;
    const feedbackText = feedbackTextarea.value.trim();
    const email = emailInput.value.trim();

    if (selectedRating < 5 && !feedbackText) {
      statusDiv.style.color = "";
      statusDiv.textContent = "Write feedback before sending.";
      return;
    }

    submitFeedback(selectedRating, feedbackText, email);
  });

  closeButton.addEventListener("click", () => {
    feedbackForm.style.display = "none";
    feedbackTextarea.value = "";
    emailInput.value = "";
    statusDiv.textContent = "";
    selectedRating = 0;
    ratingLocked = false;

    stars.forEach((s) => {
      s.textContent = "☆";
      s.classList.remove("active");
      s.classList.remove("hovered");
    });
  });
}

// Init on load
document.addEventListener("DOMContentLoaded", initFeedbackWidget);

// Re-init on page change (MkDocs Material support)
if (typeof document$ !== 'undefined') {
  document$.subscribe(() => {
    initFeedbackWidget();
  });
}
