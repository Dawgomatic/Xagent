# -*- coding: utf-8 -*-
"""
Google Search Module - Прямий парсинг Google
Пункт 3: Пошукова система з відкритими джерелами
"""
import requests
from typing import List, Dict
from bs4 import BeautifulSoup

class GoogleParser:
    """Direct Google search parsing"""
    
    def __init__(self):
        self.headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
        }
    
    def search(self, query: str, num_results: int = 5) -> List[Dict[str, str]]:
        """Alias for search_google for broader compatibility"""
        return self.search_google(query, num_results)

    def search_google(self, query: str, num_results: int = 5) -> List[Dict[str, str]]:
        """Прямий парсинг Google результатів"""
        try:
            # Google search URL
            url = f"https://www.google.com/search?q={requests.utils.quote(query)}&num={num_results}"
            
            response = requests.get(url, headers=self.headers, timeout=10)
            soup = BeautifulSoup(response.text, 'html.parser')
            
            results = []
            
            # Parse search results
            for g in soup.find_all('div', class_='g')[:num_results]:
                try:
                    # Title
                    title_elem = g.find('h3')
                    title = title_elem.text if title_elem else "No title"
                    
                    # Link
                    link_elem = g.find('a')
                    link = link_elem.get('href', '') if link_elem else ""
                    
                    # Snippet
                    snippet_elem = g.find('div', class_=['VwiC3b', 'yXK7lf'])
                    snippet = snippet_elem.text if snippet_elem else ""
                    
                    if title and link:
                        results.append({
                            'title': title,
                            'link': link,
                            'snippet': snippet[:300]
                        })
                except Exception as e:
                    continue
            
            return results
        except Exception as e:
            return [{"error": str(e)}]
    
    def format_results(self, results: List[Dict], query: str) -> str:
        """Format as Ukrainian text"""
        if not results or "error" in results[0]:
            return f"Помилка пошуку: {results[0].get('error', 'unknown')}"
        
        formatted = [f" **Google пошук:** {query}\n"]
        for i, r in enumerate(results, 1):
            formatted.append(f"{i}. **{r['title']}**")
            if r.get('snippet'):
                formatted.append(f"   {r['snippet']}")
            formatted.append(f"    {r['link']}\n")
        
        return "\n".join(formatted)
