package tech.amikos.chroma.local.core;

import com.google.gson.FieldNamingPolicy;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;

final class JsonUtil {

    static final Gson GSON = new GsonBuilder()
            .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
            .create();

    private JsonUtil() {}

    static <T> T fromJson(String json, Class<T> type) {
        return GSON.fromJson(json, type);
    }

    static String toJson(Object obj) {
        return GSON.toJson(obj);
    }
}
