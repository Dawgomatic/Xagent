/**
 * Airbnb listing data fetcher
 * Multiple source options from cheapest to most comprehensive
 *
 * SOURCE OPTIONS (ranked by cost/practicality):
 *
 * 1. AirROI (FREE + pay-as-you-go from $10)
 *    - https://airroi.com - 20M+ listings globally
 *    - FREE dashboard, pay per API call
 *    - BEST CHOICE for getting started
 *
 * 2. SearchAPI.io (pay-as-you-go, ~$0.01-0.05 per search)
 *    - https://searchapi.io/airbnb-api
 *    - Returns listing details, pricing, reviews
 *    - No subscription needed
 *
 * 
 *    - 
 *    - Gets listing details by URL or location search
 *    - Good for small batches
 *
 * 4. Airbtics ($29-99/mo)
 *    - 5M property dataset
 *    - Includes revenue estimates and occupancy
 *
 * 5. AirDNA (contact for pricing, ~$299+/mo)
 *    - Most comprehensive STR data
 *    - Overkill for initial research
 */

const axios = require('axios');

const AIRROI_KEY = process.env.AIRROI_KEY;
const RAPIDAPI_KEY = process.env.RAPIDAPI_KEY;
const SEARCHAPI_KEY = process.env.SEARCHAPI_KEY;
const AIRBTICS_KEY = process.env.AIRBTICS_KEY;

/**
 * Fetch Airbnb listings for a location
 * Auto-selects best available provider based on configured keys
 *
 * @param {string} location - ZIP code or city
 * @param {object} options
 * @returns {Array} Airbnb listing objects
 */
async function fetchAirbnbListings(location, options = {}) {
  const { maxResults = 50, checkIn, checkOut } = options;

  // Priority order: RapidAPI (free tier)
  if (RAPIDAPI_KEY) {
    return fetchViaRapidAPI(location, options);
  }
  if (AIRROI_KEY) {
    return fetchViaAirROI(location, options);
  }
  if (SEARCHAPI_KEY) {
    return fetchViaSearchAPI(location, options);
  }
  }
  if (AIRBTICS_KEY) {
    return fetchViaAirbtics(location, options);
  }

  throw new Error(
    'No Airbnb data provider configured.\n\n' +
    'Options (cheapest first):\n' +
    '  1. RapidAPI (FREE 100/mo): https://rapidapi.com/3b-data-3b-data-default/api/airbnb13 → set RAPIDAPI_KEY\n' +
    '  2. AirROI (FREE + pay-as-you-go): https://airroi.com → set AIRROI_KEY\n' +
    'Run with --demo to see the tool without API keys'
  );
}

/**
 * AirROI API - Best free option
 * Returns market-level data + individual listing data
 */
async function fetchViaAirROI(location, options = {}) {
  const { maxResults = 50 } = options;

  process.stderr.write(`  📡 Fetching Airbnb data for ${location} via AirROI...\n`);

  try {
    // AirROI market search endpoint
    const response = await axios.get('https://api.airroi.com/v1/listings/search', {
      params: {
        location,
        status: 'active',
        limit: maxResults,
        include_revenue: true
      },
      headers: {
        'Authorization': `Bearer ${AIRROI_KEY}`,
        'Content-Type': 'application/json'
      },
      timeout: 15000
    });

    const listings = response.data?.listings || response.data?.results || [];
    process.stderr.write(`  ✅ Found ${listings.length} active Airbnb listings\n`);

    return listings.map(normalizeAirROIListing);

  } catch (err) {
    if (err.response?.status === 401) {
      throw new Error('Invalid AirROI API key. Check AIRROI_KEY in .env');
    }
    throw new Error(`AirROI fetch failed: ${err.message}`);
  }
}

/**
 * RapidAPI airbnb13 - Best free option for search
 * Free: 100 requests/month, Basic: $10/mo for 500
 * Returns listings with price, rating, reviews, coordinates
 */
async function fetchViaRapidAPI(location, options = {}) {
  const { maxResults = 20, checkIn, checkOut } = options;

  // Default dates (next available weekend)
  const today = new Date();
  const nextFriday = new Date(today);
  nextFriday.setDate(today.getDate() + (5 - today.getDay() + 7) % 7 + 7);
  const nextSunday = new Date(nextFriday);
  nextSunday.setDate(nextFriday.getDate() + 2);

  const checkinDate = checkIn || nextFriday.toISOString().split('T')[0];
  const checkoutDate = checkOut || nextSunday.toISOString().split('T')[0];

  process.stderr.write(`  📡 Fetching Airbnb listings for ${location} via RapidAPI...\n`);

  try {
    const response = await axios.get('https://airbnb13.p.rapidapi.com/search-location', {
      params: {
        location,
        checkin: checkinDate,
        checkout: checkoutDate,
        adults: 2,
        currency: 'USD',
        page: 1
      },
      headers: {
        'X-RapidAPI-Key': RAPIDAPI_KEY,
        'X-RapidAPI-Host': 'airbnb13.p.rapidapi.com'
      },
      timeout: 20000
    });

    const results = response.data?.results || [];
    process.stderr.write(`  ✅ Found ${results.length} Airbnb listings via RapidAPI\n`);

    return results.slice(0, maxResults).map(normalizeRapidAPIListing);

  } catch (err) {
    if (err.response?.status === 403 || err.response?.status === 429) {
      throw new Error('RapidAPI rate limit hit or subscription needed.\n' +
        'Free tier = 100 requests/month.\n' +
        'Subscribe at: https://rapidapi.com/3b-data-3b-data-default/api/airbnb13');
    }
    if (err.response?.status === 401) {
      throw new Error('Invalid RapidAPI key. Check RAPIDAPI_KEY in .env');
    }
    throw new Error(`RapidAPI Airbnb fetch failed: ${err.message}`);
  }
}

/**
 * SearchAPI.io - Pay-per-use Airbnb scraper
 * ~$0.01-0.05 per search call
 */
async function fetchViaSearchAPI(location, options = {}) {
  const { maxResults = 50, checkIn, checkOut } = options;

  // Default dates (next available weekend) if not provided
  const today = new Date();
  const nextFriday = new Date(today);
  nextFriday.setDate(today.getDate() + (5 - today.getDay() + 7) % 7 + 7);
  const nextSunday = new Date(nextFriday);
  nextSunday.setDate(nextFriday.getDate() + 2);

  const checkinDate = checkIn || nextFriday.toISOString().split('T')[0];
  const checkoutDate = checkOut || nextSunday.toISOString().split('T')[0];

  process.stderr.write(`  📡 Fetching Airbnb listings for ${location} via SearchAPI...\n`);

  try {
    const response = await axios.get('https://www.searchapi.io/api/v1/search', {
      params: {
        engine: 'airbnb',
        q: location,
        check_in: checkinDate,
        check_out: checkoutDate,
        num_adults: 2,
        currency: 'USD',
        page: 1
      },
      headers: {
        'Authorization': `Bearer ${SEARCHAPI_KEY}`
      },
      timeout: 20000
    });

    const results = response.data?.results || response.data?.listings || [];
    process.stderr.write(`  ✅ Found ${results.length} active Airbnb listings via SearchAPI\n`);

    return results.slice(0, maxResults).map(normalizeSearchAPIListing);

  } catch (err) {
    if (err.response?.status === 401) {
      throw new Error('Invalid SearchAPI key. Check SEARCHAPI_KEY in .env');
    }
    throw new Error(`SearchAPI Airbnb fetch failed: ${err.message}`);
  }
}

/**
 * Actor: maxcopell/airbnb-scraper
 */
  const { maxResults = 50 } = options;


  const AIRBNB_ACTOR_ID = 'tri_angle~airbnb-scraper';

  try {
    const runResponse = await axios.post(
      {
        locationQueries: [location],
        maxListings: maxResults,
        currency: 'USD'
      },
      { timeout: 10000 }
    );

    const runId = runResponse.data?.data?.id;

    // Poll for results
    let attempts = 0;
    while (attempts < 24) { // up to 2 minutes
      await new Promise(r => setTimeout(r, 5000));
      attempts++;

      const statusRes = await axios.get(
        { timeout: 10000 }
      );

      const status = statusRes.data?.data?.status;
      process.stderr.write(`\r  ⏳ Airbnb scrape: ${status} (${attempts * 5}s)...`);

      if (status === 'SUCCEEDED') {
        break;
      }
      if (status === 'FAILED') {
      }
    }

    const dataRes = await axios.get(
      { timeout: 30000 }
    );

    const items = dataRes.data || [];
    process.stderr.write(`  ✅ Retrieved ${items.length} Airbnb listings\n`);


  } catch (err) {
  }
}

/**
 * Airbtics API - $29-99/mo
 */
async function fetchViaAirbtics(location, options = {}) {
  const { maxResults = 50 } = options;

  process.stderr.write(`  📡 Fetching Airbnb data for ${location} via Airbtics...\n`);

  try {
    const response = await axios.get('https://api.airbtics.com/api/v1/listings', {
      params: {
        city: location,
        limit: maxResults,
        status: 'active'
      },
      headers: {
        'X-API-Key': AIRBTICS_KEY
      },
      timeout: 15000
    });

    const listings = response.data?.data || response.data?.listings || [];
    return listings.map(normalizeAirbticListing);

  } catch (err) {
    throw new Error(`Airbtics fetch failed: ${err.message}`);
  }
}

// ─── Normalizers ──────────────────────────────────────────────────────────────

function normalizeAirROIListing(item) {
  return {
    id: item.id || item.airbnb_id,
    address: item.address || item.full_address,
    city: item.city,
    state: item.state,
    zip: item.zipcode || item.zip,
    title: item.name || item.title,
    beds: item.bedrooms,
    baths: item.bathrooms,
    maxGuests: item.accommodates || item.max_guests,
    nightly_rate: item.price || item.nightly_rate,
    monthly_revenue_avg: item.monthly_revenue || item.avg_monthly_revenue,
    annual_revenue_est: item.annual_revenue || (item.monthly_revenue * 12),
    occupancy_rate: item.occupancy_rate || item.occupancy,
    total_reviews: item.reviews_count || item.number_of_reviews,
    avg_rating: item.rating || item.review_score,
    host_status: item.is_superhost ? 'Superhost' : 'Regular',
    airbnbUrl: item.listing_url || `https://airbnb.com/rooms/${item.airbnb_id}`,
    lat: item.latitude || item.lat,
    lng: item.longitude || item.lng
  };
}

function normalizeRapidAPIListing(item) {
  // airbnb13 API response format
  const nightlyRate = item.price?.rate || item.price?.total;
  const estimatedOccupancy = 0.70;
  const monthlyRevenue = nightlyRate ? Math.round(nightlyRate * 30 * estimatedOccupancy) : null;

  return {
    id: `rapidapi-${item.id}`,
    address: item.name || item.title,
    city: item.city || '',
    state: '',
    zip: '',
    title: item.name || item.title,
    beds: item.bedrooms || item.beds,
    baths: item.bathrooms,
    maxGuests: item.persons || item.guests,
    nightly_rate: nightlyRate,
    monthly_revenue_avg: monthlyRevenue,
    annual_revenue_est: monthlyRevenue ? monthlyRevenue * 12 : null,
    occupancy_rate: estimatedOccupancy,
    total_reviews: item.reviewsCount || item.numberOfReviews,
    avg_rating: item.rating,
    host_status: item.isSuperhost ? 'Superhost' : 'Regular',
    airbnbUrl: item.url || item.deeplink || `https://airbnb.com/rooms/${item.id}`,
    lat: item.lat || item.coordinate?.latitude,
    lng: item.lng || item.coordinate?.longitude,
    roomType: item.type || item.roomType,
    images: item.images?.slice(0, 3),
    _note: monthlyRevenue ? 'Revenue estimated at 70% occupancy from nightly rate' : 'Nightly rate unavailable'
  };
}

function normalizeSearchAPIListing(item) {
  return {
    id: `searchapi-${item.id}`,
    address: `${item.name}`, // SearchAPI often doesn't return exact address
    city: item.location?.city || '',
    state: item.location?.state || '',
    zip: item.location?.zip || '',
    title: item.name,
    beds: item.beds || item.bedrooms,
    baths: item.bathrooms,
    maxGuests: item.guests,
    nightly_rate: item.price?.rate,
    monthly_revenue_avg: null, // SearchAPI doesn't include revenue estimates
    annual_revenue_est: null,
    occupancy_rate: null,
    total_reviews: item.reviews_count,
    avg_rating: item.rating,
    host_status: item.is_superhost ? 'Superhost' : 'Regular',
    airbnbUrl: item.url || `https://airbnb.com/rooms/${item.id}`,
    lat: item.coordinates?.lat,
    lng: item.coordinates?.lng,
    _note: 'Revenue data not available via SearchAPI - use AirROI for full metrics'
  };
}

  // tri_angle~airbnb-scraper output format
  const ratingObj = typeof item.rating === 'object' ? item.rating : {};
  const ratingVal = ratingObj.guestSatisfaction || ratingObj.overall || ratingObj.value || (typeof item.rating === 'number' ? item.rating : null);
  const reviewCount = ratingObj.reviewsCount || item.reviewsCount || item.numberOfReviews || 0;
  const coords = item.coordinates || item.location || {};

  // Extract nightly rate from price breakdown
  let nightlyRate = null;
  if (item.price?.breakDown?.basePrice?.description) {
    // Format: "5 nights x $212.20"
    const match = item.price.breakDown.basePrice.description.match(/\$([0-9,.]+)/);
    if (match) nightlyRate = parseFloat(match[1].replace(',', ''));
  }
  if (!nightlyRate && item.price?.price) {
    // Fallback: parse total price string "$1,242"
    const match = item.price.price.match(/\$([0-9,.]+)/);
    if (match) nightlyRate = parseFloat(match[1].replace(',', '')) / 5; // assume 5-night search
  }

  // Estimate monthly revenue from nightly rate (assume 70% occupancy baseline)
  const estimatedOccupancy = 0.70;
  const monthlyRevenue = nightlyRate ? Math.round(nightlyRate * 30 * estimatedOccupancy) : null;

  // Check superhost from host object
  const isSuperhost = item.isSuperHost || item.isSuperhost || item.host?.isSuperhost || false;

  // Clean URL (remove query params)
  let cleanUrl = item.url || `https://airbnb.com/rooms/${item.id}`;
  if (cleanUrl.includes('?')) cleanUrl = cleanUrl.split('?')[0];

  return {
    address: item.address || item.location?.address || item.locationTitle || item.title,
    city: item.city || item.location?.city || '',
    state: item.state || item.location?.state || '',
    zip: item.zip || item.location?.zip || '',
    title: item.title || item.name,
    beds: item.bedrooms || item.beds,
    baths: item.bathrooms,
    maxGuests: item.personCapacity || item.persons || item.guests,
    nightly_rate: nightlyRate,
    monthly_revenue_avg: monthlyRevenue,
    annual_revenue_est: monthlyRevenue ? monthlyRevenue * 12 : null,
    occupancy_rate: estimatedOccupancy,
    total_reviews: reviewCount,
    avg_rating: ratingVal,
    host_status: isSuperhost ? 'Superhost' : 'Regular',
    airbnbUrl: cleanUrl,
    lat: coords.latitude || coords.lat,
    lng: coords.longitude || coords.lng,
    roomType: item.roomType,
    _note: monthlyRevenue ? 'Revenue estimated at 70% occupancy from nightly rate' : 'Nightly rate unavailable'
  };
}

function normalizeAirbticListing(item) {
  return {
    id: item.airbnb_id || item.id,
    address: item.address,
    city: item.city,
    state: item.state,
    zip: item.zipcode,
    title: item.listing_name,
    beds: item.bedrooms,
    baths: item.bathrooms,
    maxGuests: item.accommodates,
    nightly_rate: item.avg_daily_rate,
    monthly_revenue_avg: item.monthly_revenue,
    annual_revenue_est: item.annual_revenue,
    occupancy_rate: item.occupancy_rate,
    total_reviews: item.reviews_count,
    avg_rating: item.review_score,
    host_status: item.is_superhost ? 'Superhost' : 'Regular',
    airbnbUrl: `https://airbnb.com/rooms/${item.airbnb_id}`,
    lat: item.lat,
    lng: item.lng
  };
}

module.exports = { fetchAirbnbListings };
