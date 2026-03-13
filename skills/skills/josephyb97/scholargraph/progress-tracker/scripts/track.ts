/**
 * Progress Tracker - Core Module
 * 进展追踪核心模块
 * 
 * 实时监控领域新动态：
 * - 关键词监控
 * - 学者追踪
 * - 会议论文追踪
 * - 生成定期报告
 */

import ZAI from 'z-ai-web-dev-sdk';
import type {
  WatchConfig,
  TrackerSettings,
  ProgressReport,
  PaperUpdate,
  TrendingTopic,
  PaperRecommendation,
  UpdateOptions,
  ReportOptions,
  TrendAnalysis
} from './types';

const DEFAULT_SETTINGS: TrackerSettings = {
  maxResultsPerWatch: 20,
  enableNotifications: false,
  reportSchedule: {
    daily: '09:00',
    weekly: 'Monday 09:00',
    monthly: '1st 09:00'
  }
};

export default class ProgressTracker {
  private zai: Awaited<ReturnType<typeof ZAI.create>> | null = null;
  private watches: WatchConfig[] = [];
  private settings: TrackerSettings = DEFAULT_SETTINGS;
  private history: Map<string, PaperUpdate[]> = new Map();

  async initialize(): Promise<void> {
    if (!this.zai) {
      this.zai = await ZAI.create();
    }
  }

  /**
   * 添加监控项
   */
  async addWatch(config: Omit<WatchConfig, 'id' | 'createdAt' | 'active'>): Promise<WatchConfig> {
    const watch: WatchConfig = {
      ...config,
      id: `watch_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      active: true,
      createdAt: new Date().toISOString()
    };

    this.watches.push(watch);
    return watch;
  }

  /**
   * 批量添加监控
   */
  async addWatches(configs: Array<Omit<WatchConfig, 'id' | 'createdAt' | 'active'>>): Promise<WatchConfig[]> {
    const added: WatchConfig[] = [];
    for (const config of configs) {
      const watch = await this.addWatch(config);
      added.push(watch);
    }
    return added;
  }

  /**
   * 移除监控
   */
  removeWatch(watchId: string): boolean {
    const index = this.watches.findIndex(w => w.id === watchId);
    if (index > -1) {
      this.watches.splice(index, 1);
      return true;
    }
    return false;
  }

  /**
   * 获取所有监控
   */
  getWatches(): WatchConfig[] {
    return [...this.watches];
  }

  /**
   * 获取最新更新
   */
  async getUpdates(options: UpdateOptions = {}): Promise<PaperUpdate[]> {
    await this.initialize();

    const { since, limit = 50, watchIds } = options;
    const allUpdates: PaperUpdate[] = [];

    // 获取要检查的监控项
    const watchesToCheck = watchIds
      ? this.watches.filter(w => watchIds.includes(w.id))
      : this.watches.filter(w => w.active);

    // 并行检查每个监控项
    const results = await Promise.all(
      watchesToCheck.map(watch => this.checkWatch(watch, since))
    );

    // 合并结果
    results.forEach(updates => {
      allUpdates.push(...updates);
    });

    // 去重
    const deduped = this.deduplicateUpdates(allUpdates);

    // 排序（按日期和重要性）
    deduped.sort((a, b) => {
      if (a.importance !== b.importance) {
        const order = { high: 0, medium: 1, low: 2 };
        return order[a.importance] - order[b.importance];
      }
      return new Date(b.publishDate).getTime() - new Date(a.publishDate).getTime();
    });

    return deduped.slice(0, limit);
  }

  /**
   * 检查单个监控项
   */
  private async checkWatch(watch: WatchConfig, since?: string): Promise<PaperUpdate[]> {
    try {
      let searchQuery = '';

      switch (watch.type) {
        case 'keyword':
          searchQuery = `${watch.value} research paper`;
          break;
        case 'author':
          searchQuery = `author:${watch.value} paper`;
          break;
        case 'conference':
          searchQuery = `${watch.value} 2024 paper`;
          break;
        default:
          searchQuery = watch.value;
      }

      const results = await this.zai!.functions.invoke('web_search', {
        query: searchQuery,
        num: this.settings.maxResultsPerWatch,
        recency_days: this.getRecencyDays(watch.frequency)
      });

      const updates: PaperUpdate[] = results.map((item: any, index: number) => ({
        id: `${watch.id}_${index}`,
        title: item.name,
        authors: [],
        publishDate: item.date || new Date().toISOString().split('T')[0],
        source: item.host_name,
        url: item.url,
        abstract: item.snippet,
        keywords: [],
        matchedWatches: [watch.value],
        relevanceScore: this.calculateRelevance(item.snippet || '', watch.value),
        importance: this.assessImportance(item),
        summary: ''
      }));

      // 过滤日期
      if (since) {
        const sinceDate = new Date(since);
        return updates.filter(u => new Date(u.publishDate) >= sinceDate);
      }

      return updates;
    } catch (error) {
      console.error(`Error checking watch ${watch.id}:`, error);
      return [];
    }
  }

  /**
   * 获取时间范围（天数）
   */
  private getRecencyDays(frequency: string): number {
    switch (frequency) {
      case 'daily': return 1;
      case 'weekly': return 7;
      case 'monthly': return 30;
      default: return 7;
    }
  }

  /**
   * 计算相关性分数
   */
  private calculateRelevance(text: string, keyword: string): number {
    const lowerText = text.toLowerCase();
    const lowerKeyword = keyword.toLowerCase();

    // 简单的相关性计算
    if (lowerText.includes(lowerKeyword)) {
      return 0.8;
    }

    // 检查部分匹配
    const words = lowerKeyword.split(' ');
    const matchCount = words.filter(w => lowerText.includes(w)).length;
    return matchCount / words.length * 0.6;
  }

  /**
   * 评估重要性
   */
  private assessImportance(item: any): 'high' | 'medium' | 'low' {
    const snippet = item.snippet || '';
    const title = item.name || '';

    // 高重要性关键词
    const highKeywords = ['breakthrough', 'new state-of-the-art', 'novel', 'first'];
    if (highKeywords.some(k => 
      title.toLowerCase().includes(k) || snippet.toLowerCase().includes(k)
    )) {
      return 'high';
    }

    // 中等重要性
    const mediumKeywords = ['improve', 'enhance', 'propose', 'introduce'];
    if (mediumKeywords.some(k => 
      title.toLowerCase().includes(k) || snippet.toLowerCase().includes(k)
    )) {
      return 'medium';
    }

    return 'low';
  }

  /**
   * 去重更新
   */
  private dededuplicateUpdates(updates: PaperUpdate[]): PaperUpdate[] {
    const seen = new Map<string, PaperUpdate>();

    for (const update of updates) {
      const normalizedTitle = update.title.toLowerCase().substring(0, 50);
      if (!seen.has(normalizedTitle)) {
        seen.set(normalizedTitle, update);
      } else {
        // 合并匹配的监控项
        const existing = seen.get(normalizedTitle)!;
        existing.matchedWatches = [...new Set([...existing.matchedWatches, ...update.matchedWatches])];
        existing.relevanceScore = Math.max(existing.relevanceScore, update.relevanceScore);
      }
    }

    return Array.from(seen.values());
  }

  /**
   * 生成报告
   */
  async generateReport(options: ReportOptions): Promise<ProgressReport> {
    await this.initialize();

    const { type, includeSummaries = true, highlightImportant = true, topic } = options;

    // 确定时间范围
    const period = this.getReportPeriod(type);

    // 获取更新
    const papers = await this.getUpdates({
      since: period.start,
      limit: 100
    });

    // 如果指定了主题，过滤
    const filteredPapers = topic
      ? papers.filter(p => 
          p.title.toLowerCase().includes(topic.toLowerCase()) ||
          p.matchedWatches.some(w => w.toLowerCase().includes(topic.toLowerCase()))
        )
      : papers;

    // 识别趋势主题
    const trending = await this.identifyTrends(filteredPapers);

    // 生成推荐
    const recommendations = this.generateRecommendations(filteredPapers, highlightImportant);

    // 生成摘要
    if (includeSummaries && filteredPapers.length > 0) {
      await this.addSummaries(filteredPapers.slice(0, 10));
    }

    // 计算摘要统计
    const summary = {
      totalPapers: filteredPapers.length,
      highlightedPapers: filteredPapers.filter(p => p.importance === 'high').length,
      newKeywords: this.extractNewKeywords(filteredPapers),
      trendingTopics: trending.map(t => t.topic),
      watchStats: this.calculateWatchStats(filteredPapers)
    };

    return {
      reportType: type,
      period,
      summary,
      papers: filteredPapers.slice(0, 50),
      trending,
      recommendations,
      generatedAt: new Date().toISOString()
    };
  }

  /**
   * 获取报告时间段
   */
  private getReportPeriod(type: string): { start: string; end: string } {
    const end = new Date();
    const start = new Date();

    switch (type) {
      case 'daily':
        start.setDate(start.getDate() - 1);
        break;
      case 'weekly':
        start.setDate(start.getDate() - 7);
        break;
      case 'monthly':
        start.setMonth(start.getMonth() - 1);
        break;
    }

    return {
      start: start.toISOString().split('T')[0],
      end: end.toISOString().split('T')[0]
    };
  }

  /**
   * 识别趋势
   */
  private async identifyTrends(papers: PaperUpdate[]): Promise<TrendingTopic[]> {
    // 提取关键词
    const keywordCounts = new Map<string, number>();

    for (const paper of papers) {
      const words = paper.title.toLowerCase().split(/\s+/);
      for (const word of words) {
        if (word.length > 4) { // 忽略短词
          keywordCounts.set(word, (keywordCounts.get(word) || 0) + 1);
        }
      }
    }

    // 转换为趋势主题
    const trending: TrendingTopic[] = [];
    const sortedKeywords = [...keywordCounts.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5);

    for (const [topic, count] of sortedKeywords) {
      const relatedPapers = papers
        .filter(p => p.title.toLowerCase().includes(topic))
        .slice(0, 3)
        .map(p => p.title);

      trending.push({
        topic,
        paperCount: count,
        changePercent: 0, // 需要历史数据计算
        keyPapers: relatedPapers,
        trend: 'rising'
      });
    }

    return trending;
  }

  /**
   * 生成推荐
   */
  private generateRecommendations(papers: PaperUpdate[], highlightImportant: boolean): PaperRecommendation[] {
    const recommendations: PaperRecommendation[] = [];

    // 高重要性论文优先推荐
    const sortedPapers = highlightImportant
      ? [...papers].sort((a, b) => {
          const order = { high: 0, medium: 1, low: 2 };
          return order[a.importance] - order[b.importance];
        })
      : papers;

    for (const paper of sortedPapers.slice(0, 10)) {
      recommendations.push({
        paper,
        reason: paper.importance === 'high'
          ? '高重要性论文，可能代表重要突破'
          : paper.matchedWatches.length > 1
            ? '匹配多个监控主题'
            : '与您的关注领域相关',
        priority: paper.importance === 'high' ? 1 : paper.importance === 'medium' ? 2 : 3
      });
    }

    return recommendations;
  }

  /**
   * 添加AI摘要
   */
  private async addSummaries(papers: PaperUpdate[]): Promise<void> {
    const titles = papers.map(p => p.title).join('\n- ');

    const completion = await this.zai!.chat.completions.create({
      messages: [
        {
          role: 'system',
          content: '你是一位研究文献专家，能够为学术论文生成简洁的摘要。'
        },
        {
          role: 'user',
          content: `为以下论文生成一句话摘要（每行一个）:\n- ${titles}`
        }
      ],
      temperature: 0.3
    });

    const summaries = (completion.choices[0]?.message?.content || '').split('\n').filter(Boolean);

    papers.forEach((paper, index) => {
      if (summaries[index]) {
        paper.summary = summaries[index].replace(/^-\s*/, '');
      }
    });
  }

  /**
   * 提取新关键词
   */
  private extractNewKeywords(papers: PaperUpdate[]): string[] {
    const keywords = new Set<string>();

    for (const paper of papers) {
      // 从标题提取潜在关键词
      const words = paper.title.match(/\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b/g) || [];
      words.forEach(w => keywords.add(w));
    }

    return [...keywords].slice(0, 10);
  }

  /**
   * 计算监控统计
   */
  private calculateWatchStats(papers: PaperUpdate[]): Array<{ watchId: string; watchValue: string; matchCount: number }> {
    const stats = new Map<string, { value: string; count: number }>();

    for (const paper of papers) {
      for (const watchValue of paper.matchedWatches) {
        const existing = stats.get(watchValue);
        if (existing) {
          existing.count++;
        } else {
          stats.set(watchValue, { value: watchValue, count: 1 });
        }
      }
    }

    return [...stats.entries()].map(([watchId, data]) => ({
      watchId,
      watchValue: data.value,
      matchCount: data.count
    }));
  }

  /**
   * 分析趋势
   */
  async analyzeTrends(options: { topic: string; timeframe: 'week' | 'month' | 'quarter' }): Promise<TrendAnalysis> {
    await this.initialize();

    const { topic, timeframe } = options;

    // 搜索相关论文
    const results = await this.zai!.functions.invoke('web_search', {
      query: `${topic} research paper`,
      num: 20,
      recency_days: timeframe === 'week' ? 7 : timeframe === 'month' ? 30 : 90
    });

    const papers: PaperUpdate[] = results.map((item: any, index: number) => ({
      id: `trend_${index}`,
      title: item.name,
      authors: [],
      publishDate: item.date || '',
      source: item.host_name,
      url: item.url,
      abstract: item.snippet,
      keywords: [],
      matchedWatches: [topic],
      relevanceScore: 0.5,
      importance: 'medium'
    }));

    return {
      topic,
      timeframe,
      paperCount: papers.length,
      previousCount: 0,
      changePercent: 0,
      topPapers: papers.slice(0, 5),
      emergingKeywords: [],
      decliningKeywords: []
    };
  }

  /**
   * 导出报告为Markdown
   */
  toMarkdown(report: ProgressReport): string {
    const md = `#  ${report.reportType === 'daily' ? '日报' : report.reportType === 'weekly' ? '周报' : '月报'}

**时间段**: ${report.period.start} ~ ${report.period.end}

##  概览

- 总论文数: ${report.summary.totalPapers}
- 高重要性: ${report.summary.highlightedPapers}
- 趋势主题: ${report.summary.trendingTopics.slice(0, 5).join(', ')}

##  趋势主题

${report.trending.map(t => `
### ${t.topic}
- 论文数: ${t.paperCount}
- 代表论文: ${t.keyPapers.slice(0, 2).join(', ')}
`).join('\n')}

##  重点论文

${report.papers.filter(p => p.importance === 'high').slice(0, 5).map(p => `
### ${p.title}
- 来源: ${p.source}
- 日期: ${p.publishDate}
- 链接: ${p.url}
${p.summary ? `- 摘要: ${p.summary}` : ''}
`).join('\n')}

##  推荐

${report.recommendations.slice(0, 5).map(r => `
- **${r.paper.title}**
  - 推荐理由: ${r.reason}
  - 链接: ${r.paper.url}
`).join('\n')}

---
*生成时间: ${report.generatedAt}*
`;

    return md;
  }
}

// CLI 支持
if (import.meta.main) {
  const args = process.argv.slice(2);
  const command = args[0];

  const tracker = new ProgressTracker();

  if (command === 'add') {
    const type = args[1] as 'keyword' | 'author' | 'conference';
    const value = args[2];
    const frequencyIndex = args.indexOf('--frequency');
    const frequency = frequencyIndex > -1 ? args[frequencyIndex + 1] as any : 'daily';

    tracker.initialize().then(() => 
      tracker.addWatch({ type, value, frequency })
    ).then(watch => {
      console.log('Watch added:', watch);
    });
  } else if (command === 'report') {
    const typeIndex = args.indexOf('--type');
    const type = typeIndex > -1 ? args[typeIndex + 1] as any : 'daily';
    const outputIndex = args.indexOf('--output');
    const outputFile = outputIndex > -1 ? args[outputIndex + 1] : null;

    tracker.initialize().then(() => 
      tracker.generateReport({ type })
    ).then(report => {
      if (outputFile) {
        const fs = require('fs');
        fs.writeFileSync(outputFile, tracker.toMarkdown(report));
        console.log(`Report saved to ${outputFile}`);
      } else {
        console.log(JSON.stringify(report, null, 2));
      }
    });
  } else {
    console.error('Usage: track.ts add <type> <value> [--frequency daily|weekly|monthly]');
    console.error('       track.ts report --type daily|weekly|monthly [--output <file>]');
    process.exit(1);
  }
}
