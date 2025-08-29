// Main JavaScript file for openapi-mcp website

document.addEventListener('DOMContentLoaded', function() {
  // Mobile menu toggle functionality
  const mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
  const navbar = document.querySelector('.navbar');
  
  if (mobileMenuToggle) {
    mobileMenuToggle.addEventListener('click', function() {
      document.body.classList.toggle('mobile-nav-open');
    });
  }
  
  // Add active class to current nav link
  const currentPath = window.location.pathname;
  const navLinks = document.querySelectorAll('.nav-links a');
  
  navLinks.forEach(link => {
    const linkPath = link.getAttribute('href');
    if (currentPath === linkPath || 
        (linkPath !== '/' && currentPath.startsWith(linkPath))) {
      link.classList.add('active');
    }
  });
  
  // Initialize code highlighting if highlight.js is loaded
  if (typeof hljs !== 'undefined') {
    document.querySelectorAll('pre code').forEach((block) => {
      hljs.highlightBlock(block);
    });
  }
  
  // Smooth scrolling for anchor links
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
      e.preventDefault();
      
      const targetId = this.getAttribute('href').substring(1);
      const targetElement = document.getElementById(targetId);
      
      if (targetElement) {
        window.scrollTo({
          top: targetElement.offsetTop - 80, // Offset for fixed header
          behavior: 'smooth'
        });
      }
    });
  });
  
  // Add copy functionality to code blocks
  document.querySelectorAll('pre').forEach(block => {
    const copyButton = document.createElement('button');
    copyButton.className = 'copy-button';
    copyButton.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/><path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/></svg>';
    copyButton.style.position = 'absolute';
    copyButton.style.top = '0.5rem';
    copyButton.style.right = '0.5rem';
    copyButton.style.padding = '0.25rem';
    copyButton.style.background = 'rgba(0, 0, 0, 0.3)';
    copyButton.style.border = 'none';
    copyButton.style.borderRadius = '4px';
    copyButton.style.color = 'white';
    copyButton.style.cursor = 'pointer';
    
    // Make the pre position relative to position the button
    block.style.position = 'relative';
    
    block.appendChild(copyButton);
    
    copyButton.addEventListener('click', () => {
      const code = block.querySelector('code') || block;
      const text = code.innerText;
      
      navigator.clipboard.writeText(text).then(() => {
        copyButton.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M10.97 4.97a.75.75 0 0 1 1.07 1.05l-3.99 4.99a.75.75 0 0 1-1.08.02L4.324 8.384a.75.75 0 1 1 1.06-1.06l2.094 2.093 3.473-4.425a.267.267 0 0 1 .02-.022z"/></svg>';
        
        setTimeout(() => {
          copyButton.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/><path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/></svg>';
        }, 2000);
      }).catch(err => {
        console.error('Failed to copy text: ', err);
      });
    });
  });
  
  // Handle documentation sidebar (if exists)
  const docsSidebar = document.querySelector('.docs-sidebar');
  if (docsSidebar) {
    const docsLinks = docsSidebar.querySelectorAll('a');
    const currentDocPath = window.location.pathname;
    
    docsLinks.forEach(link => {
      const linkPath = link.getAttribute('href');
      if (currentDocPath === linkPath) {
        link.classList.add('active');
        
        // Expand parent sections if any
        const parentLi = link.closest('li.has-submenu');
        if (parentLi) {
          parentLi.classList.add('expanded');
        }
      }
    });
    
    // Toggle submenu items
    const submenuToggles = docsSidebar.querySelectorAll('.submenu-toggle');
    submenuToggles.forEach(toggle => {
      toggle.addEventListener('click', function(e) {
        e.preventDefault();
        const li = this.closest('li');
        li.classList.toggle('expanded');
      });
    });
  }
});