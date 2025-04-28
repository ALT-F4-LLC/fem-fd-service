document.addEventListener('DOMContentLoaded', function() {
    const cheerButtons = document.querySelectorAll('.cheer-button');
    
    cheerButtons.forEach(button => {
        button.addEventListener('click', function() {
            const updateId = this.dataset.updateId;
            const isCheer = this.classList.contains('cheers');
            const url = isCheer ? '/like' : '/unlike';

            fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `update_id=${updateId}`,
            })
            .then(response => {
                if (response.ok) {
                    // Toggle button text and class
                    const buttonText = isCheer ? 'Uncheer' : 'Cheer';
                    
                    this.innerHTML = `🎉 ${buttonText}`;
                    
                    this.classList.toggle('cheers');
                    this.classList.toggle('uncheers');

                    // Update cheer count
                    const cheerCountSpan = this.parentElement.querySelector('.cheer-count');
                    if (cheerCountSpan) {
                        let count = parseInt(cheerCountSpan.textContent);
                        count = isCheer ? count + 1 : count - 1;
                        cheerCountSpan.textContent = `${count} Cheers`;
                    } else if (isCheer) {
                        // Create new cheer count span if it doesn't exist
                        const newCheerCountSpan = document.createElement('span');
                        newCheerCountSpan.className = 'cheer-count';
                        newCheerCountSpan.textContent = '1 Cheer';
                        this.parentElement.insertBefore(newCheerCountSpan, this);
                    }
                }
            })
            .catch(error => console.error('Error:', error));
        });
    });

    // Follow/Unfollow functionality
    const followButton = document.getElementById('follow-button');
    const unfollowButton = document.getElementById('unfollow-button');

    if (followButton) {
        followButton.addEventListener('click', function() {
            const username = this.dataset.username;
            followUnfollowAction('/follow', username, this);
        });
    }

    if (unfollowButton) {
        unfollowButton.addEventListener('click', function() {
            const username = this.dataset.username;
            followUnfollowAction('/unfollow', username, this);
        });
    }

    function followUnfollowAction(url, username, button) {
        fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `username=${username}`,
        })
        .then(response => {
            if (response.ok) {
                // Toggle button
                const isNowFollowing = button.id === 'follow-button';
                const newButton = document.createElement('button');
                newButton.id = isNowFollowing ? 'unfollow-button' : 'follow-button';
                newButton.dataset.username = username;
                newButton.innerHTML = isNowFollowing ? 'Unfollow' : 'Follow';
                newButton.addEventListener('click', function() {
                    followUnfollowAction(isNowFollowing ? '/unfollow' : '/follow', username, this);
                });
                button.parentNode.replaceChild(newButton, button);

                // Update follower count
                const followerCountSpan = document.querySelector('.followers');
                if (followerCountSpan) {
                    let count = parseInt(followerCountSpan.textContent);
                    count = isNowFollowing ? count + 1 : count - 1;
                    followerCountSpan.textContent = `${count} Followers`;
                }
            }
        })
        .catch(error => console.error('Error:', error));
    }

  // Auto-grow functionality for textareas
  const textareas = document.querySelectorAll('textarea');
  textareas.forEach(textarea => {
      textarea.addEventListener('input', autoGrow);
      // Initial call to set the height
      autoGrow.call(textarea);
  });

  function autoGrow() {
      this.style.height = 'auto';
      this.style.height = (this.scrollHeight) + 'px';
  }
});