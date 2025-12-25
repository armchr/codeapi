package com.example.petclinic.repository;

import com.example.petclinic.model.PetType;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import java.util.Optional;

/**
 * Repository for PetType entity.
 */
@Repository
public interface PetTypeRepository extends JpaRepository<PetType, Long> {

    /**
     * Find pet type by name.
     */
    Optional<PetType> findByName(String name);

    /**
     * Check if pet type exists by name.
     */
    boolean existsByName(String name);
}
