package dude.morrildl.providence;

import java.io.ByteArrayInputStream;
import java.io.StringReader;
import java.security.KeyStore;
import java.util.Properties;

import android.content.Context;
import android.content.SharedPreferences;
import android.util.Base64;
import android.util.Log;

/**
 * Global configuration store. For ease of use, callers don't need to catch any
 * exceptions, but if the class was not initialized by somebody, the getters
 * throw runtime exceptions. Can only be initialized once, via setFrom().
 */
public class Config {
    private static Config instance;

    public static Config getInstance(Context context) {
        if (instance == null) {
            synchronized (Config.class) {
                if (instance == null) {
                    instance = new Config(context);
                }
            }
        }
        return instance;
    }

    private String canonicalServerName;
    private Context context;
    private KeyStore keystore;
    private String oAuthAudience;
    private String photoUrlBase;
    private boolean ready = false;
    private String regIdUrl;

    private Config(Context context) {
        this.context = context;
        loadConfig();
    }

    public String getCanonicalServerName() {
        readyOrThrow();
        return canonicalServerName;
    }

    public KeyStore getKeystore() {
        readyOrThrow();
        return keystore;
    }

    public String getOAuthAudience() {
        readyOrThrow();
        return oAuthAudience;
    }

    public String getPhotoUrlBase() {
        readyOrThrow();
        return photoUrlBase;
    }

    public String getRegIdUrl() {
        readyOrThrow();
        return regIdUrl;
    }

    public boolean isReady() {
        if (!ready)
            ready = loadConfig();
        return ready;
    }

    public static void storeConfig(Context context, String rawProperties) {
        context.getSharedPreferences("globalconfig", Context.MODE_PRIVATE)
                .edit().putString("properties", rawProperties).commit();
    }

    private boolean loadConfig() {
        if (ready) {
            return true;
        }

        // first see if we have a local config copy, and use it
        SharedPreferences prefs = context.getApplicationContext()
                .getSharedPreferences("globalconfig", Context.MODE_PRIVATE);
        String rawProps = prefs.getString("properties", "");
        if (!prefs.contains("properties") || "".equals(rawProps)) {
            prefs.edit().clear().commit();
            ready = false;
            return false;
        }

        Properties props = new Properties();
        try {
            props.load(new StringReader(rawProps));
        } catch (Exception e) {
            Log.e("Config.loadConfig", "exception parsing properties", e);
            prefs.edit().clear().commit();
            ready = false;
            return false;
        }

        oAuthAudience = props.getProperty("OAUTH_AUDIENCE", "");
        regIdUrl = props.getProperty("REGID_URL", "");
        photoUrlBase = props.getProperty("PHOTO_BASE", "");
        canonicalServerName = props.getProperty("CANONICAL_SERVER_NAME", "");

        if ("".equals(oAuthAudience) || "".equals(regIdUrl)
                || "".equals(photoUrlBase) || "".equals(canonicalServerName)) {
            Log.e("Config.loadConfig", "invalid or incomplete properties block");
            prefs.edit().clear().commit();
            ready = false;
            return false;
        }

        try {
            char[] ksPass = props.getProperty("KEYSTORE_PASSWORD")
                    .toCharArray();
            byte[] ksBytes = Base64.decode(props.getProperty("KEYSTORE"), 0);
            keystore = KeyStore.getInstance("BKS");
            keystore.load(new ByteArrayInputStream(ksBytes), ksPass);
        } catch (Exception e) {
            Log.e("Config.loadConfig", "error parsing keystore", e);
            prefs.edit().clear().commit();
            ready = false;
            return false;
        }

        ready = true;
        return true;
    }

    private void readyOrThrow() {
        if (!ready) {
            if (!loadConfig()) {
                throw new IllegalStateException("Config is uninitialized");
            }
        }
    }
}
